// Copyright 2023 Chainguard, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package apk

import (
	"bytes"
	"crypto/md5" //nolint:gosec // this is just for testing, md5 is good enough
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	apkfs "github.com/chainguard-dev/go-apk/pkg/fs"
)

const (
	testPrimaryPkgDir   = "testdata/alpine-316"
	testAlternatePkgDir = "testdata/alpine-317"
)

type testLocalTransport struct {
	fail         bool
	root         string
	basenameOnly bool
	headers      map[string][]string
}

func (t *testLocalTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if t.fail {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(bytes.NewBuffer([]byte("not found"))),
		}, nil
	}
	var target string
	if t.basenameOnly {
		target = filepath.Join(t.root, filepath.Base(request.URL.Path))
	} else {
		target = filepath.Join(t.root, request.URL.Path)
	}
	// we generate an etag from the md5 sum of the file contents, because it is faster and simpler
	// and the sha256 guarantees do not matter to us
	file, err := os.Open(target)
	if err != nil {
		return &http.Response{StatusCode: 404}, nil
	}
	defer file.Close()

	hash := md5.New() //nolint:gosec // this is just for testing, md5 is good enough
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("unable to calculate md5 sum of file %s: %w", target, err)
	}
	etag, err := hex.EncodeToString(hash.Sum(nil)), nil
	if err != nil {
		return nil, fmt.Errorf("unable to encode md5 sum of file %s: %w", target, err)
	}

	f, err := os.Open(target)
	if err != nil {
		return &http.Response{StatusCode: 404}, nil
	}
	headers := make(map[string][]string)
	for k, v := range t.headers {
		headers[k] = v
	}
	headers[http.CanonicalHeaderKey("etag")] = []string{etag}
	return &http.Response{
		StatusCode: 200,
		Body:       f,
		Header:     headers,
	}, nil
}

func testGetTestAPK() (*APK, apkfs.FullFS, error) {
	// load it all into memory so that we don't change any of our test data
	src := apkfs.NewMemFS()
	filesystem := os.DirFS("testdata/root")
	if walkErr := fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return src.MkdirAll(path, d.Type())
		}
		r, err := filesystem.Open(path)
		if err != nil {
			return err
		}
		w, err := src.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, d.Type())
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		return err
	}); walkErr != nil {
		return nil, nil, walkErr
	}
	apk, err := New(WithFS(src), WithIgnoreMknodErrors(ignoreMknodErrors))
	if err != nil {
		return nil, nil, err
	}
	return apk, src, err
}
