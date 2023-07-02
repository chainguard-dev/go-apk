// Copyright 2023 Chainguard, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apk

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.alpinelinux.org/alpine/go/repository"
)

type testDirEntry struct {
	path    string
	perms   os.FileMode
	dir     bool
	content []byte
	xattrs  map[string][]byte
}

func TestInstallAPKFiles(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		apk, src, err := testGetTestAPK()
		require.NoErrorf(t, err, "failed to get test APK")

		// create a tgz stream with our files
		entries := []testDirEntry{
			// do the dirs first so we are assured they go in before files
			{"etc", 0o755, true, nil, nil},
			{"etc/foo", 0o755, true, nil, nil},
			{"var", 0o755, true, nil, nil},
			{"var/lib", 0o755, true, nil, nil},
			{"var/lib/test", 0o755, true, nil, nil},

			{"etc/foo/bar", 0644, false, []byte("hello world"), nil},
			{"var/lib/test/foobar", 0644, false, []byte("hello var/lib"), nil},
			{"etc/other", 0644, false, []byte("first"), nil},
		}

		r := testCreateTGZForPackage(entries)
		headers, err := apk.installAPKFiles(context.Background(), r, "", "")
		require.NoError(t, err)

		require.Equal(t, len(headers), len(entries))

		// compare each one to make sure it is in the returned list
		headerMap := map[string]tar.Header{}
		for _, h := range headers {
			headerMap[h.Name] = h
		}
		for _, e := range entries {
			name := e.path
			h, ok := headerMap[name]
			if e.dir {
				require.True(t, ok, "directory %s not found in headers", name)
				require.Equal(t, tar.TypeDir, rune(h.Typeflag), "mismatched file type for %s", name)
				require.Equal(t, int64(e.perms), h.Mode, "mismatched permissions for %s", name)
			} else {
				require.True(t, ok, "file %s not found in headers", name)
				require.Equal(t, tar.TypeReg, rune(h.Typeflag), "mismatched file type for %s", name)
				require.Equal(t, h.Mode, int64(e.perms), "mismatched permissions for %s", name)
				require.Equal(t, int64(len(e.content)), h.Size, "mismatched size for %s", name)
			}
			delete(headerMap, name)
		}

		// compare each one in the memfs filesystem to make sure it was installed correctly
		for _, e := range entries {
			name := e.path
			fi, err := fs.Stat(src, name)
			require.NoError(t, err, "error statting %s", name)
			if e.dir {
				require.True(t, fi.IsDir(), "expected %s to be a directory, got %v", name, fi.Mode())
				require.Equal(t, fi.Mode(), os.ModeDir|e.perms, "expected %s to have permissions %v, got %v", name, e.perms, fi.Mode())
			} else {
				require.True(t, fi.Mode().IsRegular(), "expected %s to be a regular file, got %v", name, fi.Mode())
				require.Equal(t, fi.Mode(), e.perms, "expected %s to have permissions %v, got %v", name, e.perms, fi.Mode())
				require.Equal(t, fi.Size(), int64(len(e.content)), "expected %s to have size %d, got %d", name, len(e.content), fi.Size())
				actual, err := src.ReadFile(name)
				require.NoError(t, err, "error reading %s", name)
				require.True(t, bytes.Equal(actual, e.content), "unexpected content for %s: expected %q, got %q", name, e.content, actual)
			}
		}
	})

	t.Run("xattrs", func(t *testing.T) {
		apk, src, err := testGetTestAPK()
		require.NoErrorf(t, err, "failed to get test APK")

		// create a tgz stream with our files
		entries := []testDirEntry{
			// do the dirs first so we are assured they go in before files
			{"etc", 0o755, true, nil, map[string][]byte{"user.etc": []byte("hello world")}},
			{"etc/foo", 0o644, false, []byte("hello world"), map[string][]byte{"user.file": []byte("goodbye now")}},
		}

		r := testCreateTGZForPackage(entries)
		headers, err := apk.installAPKFiles(context.Background(), r, "", "")
		require.NoError(t, err)

		require.Equal(t, len(headers), len(entries))

		// compare each one to make sure it is in the returned list
		headerMap := map[string]tar.Header{}
		for _, h := range headers {
			headerMap[h.Name] = h
		}
		for _, e := range entries {
			name := e.path
			h, ok := headerMap[name]
			require.True(t, ok, "target %s not found in headers", name)
			for k, v := range e.xattrs {
				val, ok := h.PAXRecords[fmt.Sprintf("%s%s", xattrTarPAXRecordsPrefix, k)]
				require.True(t, ok, "xattr %s not found in headers for %s", k, name)
				require.Equal(t, val, string(v), "mismatched xattr %s for %s", k, name)
			}
		}

		// compare each one in the memfs filesystem to make sure it was installed correctly
		for _, e := range entries {
			name := e.path
			xattrs, err := src.ListXattrs(name)
			require.NoError(t, err, "error getting xattrs %s", name)
			require.Equal(t, len(xattrs), len(e.xattrs), "mismatched number of xattrs for %s", name)
			for k, v := range e.xattrs {
				require.Equal(t, v, xattrs[k], "mismatched xattr %s for %s", k, name)
			}
		}
	})

	t.Run("overlapping files", func(t *testing.T) {
		t.Run("different origin and content", func(t *testing.T) {
			apk, src, err := testGetTestAPK()
			require.NoErrorf(t, err, "failed to get test APK")
			// install a file in a known location
			originalContent := []byte("hello world")
			finalContent := []byte("extra long I am here")
			overwriteFilename := "etc/doublewrite" //nolint:goconst

			pkg := &repository.Package{Name: "first", Origin: "first"}

			entries := []testDirEntry{
				{"etc", 0o755, true, nil, nil},
				{overwriteFilename, 0o755, false, originalContent, nil},
			}

			r := testCreateTGZForPackage(entries)
			headers, err := apk.installAPKFiles(context.Background(), r, pkg.Origin, "")
			require.NoError(t, err)
			err = apk.addInstalledPackage(pkg, headers)
			require.NoError(t, err)

			actual, err := src.ReadFile(overwriteFilename)
			require.NoError(t, err, "error reading %s", overwriteFilename)
			require.Equal(t, originalContent, actual)

			entries = []testDirEntry{
				{overwriteFilename, 0o755, false, finalContent, nil},
			}

			r = testCreateTGZForPackage(entries)
			_, err = apk.installAPKFiles(context.Background(), r, "second", "")
			require.Error(t, err, "some double-write error")

			actual, err = src.ReadFile(overwriteFilename)
			require.NoError(t, err, "error reading %s", overwriteFilename)
			require.Equal(t, originalContent, actual)
		})
		t.Run("different origin and content, but with replaces", func(t *testing.T) {
			apk, src, err := testGetTestAPK()
			require.NoErrorf(t, err, "failed to get test APK")
			// install a file in a known location
			originalContent := []byte("hello world")
			finalContent := []byte("extra long I am here")
			overwriteFilename := "etc/doublewrite"

			pkg := &repository.Package{Name: "first", Origin: "first"}

			entries := []testDirEntry{
				{"etc", 0755, true, nil, nil},
				{overwriteFilename, 0755, false, originalContent, nil},
			}

			r := testCreateTGZForPackage(entries)
			headers, err := apk.installAPKFiles(context.Background(), r, pkg.Origin, "")
			require.NoError(t, err)
			err = apk.addInstalledPackage(pkg, headers)
			require.NoError(t, err)

			actual, err := src.ReadFile(overwriteFilename)
			require.NoError(t, err, "error reading %s", overwriteFilename)
			require.Equal(t, originalContent, actual)

			entries = []testDirEntry{
				{overwriteFilename, 0755, false, finalContent, nil},
			}

			r = testCreateTGZForPackage(entries)
			_, err = apk.installAPKFiles(context.Background(), r, "second", "first")
			require.NoError(t, err)

			actual, err = src.ReadFile(overwriteFilename)
			require.NoError(t, err, "error reading %s", overwriteFilename)
			require.Equal(t, finalContent, actual)
		})
		t.Run("same origin", func(t *testing.T) {
			apk, src, err := testGetTestAPK()
			require.NoErrorf(t, err, "failed to get test APK")
			// install a file in a known location
			originalContent := []byte("hello world")
			finalContent := []byte("extra long I am here")
			overwriteFilename := "etc/doublewrite"

			entries := []testDirEntry{
				{"etc", 0o755, true, nil, nil},
				{overwriteFilename, 0o755, false, originalContent, nil},
			}
			pkg := &repository.Package{Name: "first", Origin: "first"}

			r := testCreateTGZForPackage(entries)
			headers, err := apk.installAPKFiles(context.Background(), r, pkg.Origin, "")
			require.NoError(t, err)
			err = apk.addInstalledPackage(pkg, headers)
			require.NoError(t, err)

			actual, err := src.ReadFile(overwriteFilename)
			require.NoError(t, err, "error reading %s", overwriteFilename)
			require.Equal(t, originalContent, actual)

			entries = []testDirEntry{
				{overwriteFilename, 0o755, false, finalContent, nil},
			}

			r = testCreateTGZForPackage(entries)
			_, err = apk.installAPKFiles(context.Background(), r, pkg.Origin, "")
			require.NoError(t, err)

			actual, err = src.ReadFile(overwriteFilename)
			require.NoError(t, err, "error reading %s", overwriteFilename)
			require.Equal(t, finalContent, actual)
		})
		t.Run("different origin with same content", func(t *testing.T) {
			apk, src, err := testGetTestAPK()
			require.NoErrorf(t, err, "failed to get test APK")
			// install a file in a known location
			originalContent := []byte("hello world")
			overwriteFilename := "etc/doublewrite"

			pkg := &repository.Package{Name: "first", Origin: "first"}

			entries := []testDirEntry{
				{"etc", 0o755, true, nil, nil},
				{overwriteFilename, 0o755, false, originalContent, nil},
			}

			r := testCreateTGZForPackage(entries)
			headers, err := apk.installAPKFiles(context.Background(), r, pkg.Origin, "")
			require.NoError(t, err)
			err = apk.addInstalledPackage(pkg, headers)
			require.NoError(t, err)

			actual, err := src.ReadFile(overwriteFilename)
			require.NoError(t, err, "error reading %s", overwriteFilename)
			require.Equal(t, originalContent, actual)

			entries = []testDirEntry{
				{overwriteFilename, 0o755, false, originalContent, nil},
			}

			r = testCreateTGZForPackage(entries)
			_, err = apk.installAPKFiles(context.Background(), r, "second", "")
			require.NoError(t, err)

			actual, err = src.ReadFile(overwriteFilename)
			require.NoError(t, err, "error reading %s", overwriteFilename)
			require.Equal(t, originalContent, actual)
		})
	})
}

func testCreateTGZForPackage(entries []testDirEntry) io.Reader {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		var header *tar.Header
		if e.dir {
			header = &tar.Header{
				Name:     e.path,
				Typeflag: tar.TypeDir,
				Mode:     int64(e.perms),
			}
		} else {
			header = &tar.Header{
				Name:     e.path,
				Typeflag: tar.TypeReg,
				Mode:     int64(e.perms),
				Size:     int64(len(e.content)),
			}
		}

		if e.xattrs != nil {
			header.Format = tar.FormatPAX
			if header.PAXRecords == nil {
				header.PAXRecords = make(map[string]string)
			}
			for k, v := range e.xattrs {
				header.PAXRecords[fmt.Sprintf("%s%s", xattrTarPAXRecordsPrefix, k)] = string(v)
			}
		}

		err := tw.WriteHeader(header)
		if err != nil {
			panic(err)
		}
		if e.content != nil {
			_, _ = tw.Write(e.content)
		}
	}
	tw.Close()
	gw.Close()
	return bytes.NewReader(buf.Bytes())
}
