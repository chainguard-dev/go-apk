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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gitlab.alpinelinux.org/alpine/go/pkg/repository"
)

// cache
type cache struct {
	dir     string
	offline bool
}

// client return an http.Client that knows how to read from and write to the cache
// key is in the implementation of https://pkg.go.dev/net/http#RoundTripper
func (c cache) client(wrapped *http.Client, etagRequired bool) *http.Client {
	return &http.Client{
		Transport: &cacheTransport{
			wrapped:      wrapped,
			root:         c.dir,
			offline:      c.offline,
			etagRequired: etagRequired,
		},
	}
}

type cacheTransport struct {
	wrapped      *http.Client
	root         string
	offline      bool
	etagRequired bool
}

func (t *cacheTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// do we have the file in the cache?
	if request.URL == nil {
		return nil, fmt.Errorf("no URL in request")
	}
	cacheFile, err := cachePathFromURL(t.root, *request.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid cache path based on URL: %w", err)
	}

	if !t.etagRequired {
		// We don't cache the response for these because they get cached later in cachePackage.

		// Try to open the file in the cache.
		// If we hit an error, just send the request.
		f, err := os.Open(cacheFile)
		if err != nil {
			if t.offline {
				return nil, fmt.Errorf("failed to read %q in offline cache: %w", cacheFile, err)
			}
			return t.wrapped.Do(request)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       f,
		}, nil
	}

	if t.offline {
		cacheDir := cacheDirFromFile(cacheFile)
		des, err := os.ReadDir(cacheDir)
		if err != nil {
			return nil, fmt.Errorf("listing %q for offline cache: %w", cacheDir, err)
		}

		if len(des) == 0 {
			return nil, fmt.Errorf("no offline cached entries for %s", cacheDir)
		}

		newest, err := des[0].Info()
		if err != nil {
			return nil, err
		}

		for _, de := range des[1:] {
			fi, err := de.Info()
			if err != nil {
				return nil, err
			}

			if fi.ModTime().After(newest.ModTime()) {
				newest = fi
			}
		}

		f, err := os.Open(filepath.Join(cacheDir, newest.Name()))
		if err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode:    http.StatusOK,
			Body:          f,
			ContentLength: newest.Size(),
		}, nil
	}

	resp, err := t.wrapped.Head(request.URL.String())
	if err != nil || resp.StatusCode != 200 {
		return resp, err
	}
	initialEtag, ok := etagFromResponse(resp)
	if !ok {
		// If the server doesn't return etags, and we require them,
		// then do not cache.
		return t.wrapped.Do(request)
	}
	// We simulate content-based addressing with the etag values using an .etag
	// file extension.
	etagFile := cacheFileFromEtag(cacheFile, initialEtag)
	f, err := os.Open(etagFile)
	if err != nil {
		return t.retrieveAndSaveFile(request, func(r *http.Response) (string, error) {
			// On the etag path, use the etag from the actual response to
			// compute the final file name.
			finalEtag, ok := etagFromResponse(r)
			if !ok {
				return "", fmt.Errorf("GET response did not contain an etag, but HEAD returned %q", initialEtag)
			}

			return cacheFileFromEtag(cacheFile, finalEtag), nil
		})
	}
	return &http.Response{
		StatusCode:    http.StatusOK,
		Body:          f,
		ContentLength: resp.ContentLength,
	}, nil
}

func cacheDirFromFile(cacheFile string) string {
	if strings.HasSuffix(cacheFile, "APKINDEX.tar.gz") {
		return filepath.Join(filepath.Dir(cacheFile), "APKINDEX")
	}

	return filepath.Dir(cacheFile)
}

func cacheFileFromEtag(cacheFile, etag string) string {
	cacheDir := filepath.Dir(cacheFile)
	ext := ".etag"

	// Keep all the index files under APKINDEX/ with appropriate file extension.
	if strings.HasSuffix(cacheFile, "APKINDEX.tar.gz") {
		cacheDir = filepath.Join(cacheDir, "APKINDEX")
		ext = ".tar.gz"
	}

	return filepath.Join(cacheDir, etag+ext)
}

func etagFromResponse(resp *http.Response) (string, bool) {
	remoteEtag, ok := resp.Header[http.CanonicalHeaderKey("etag")]
	if !ok || len(remoteEtag) == 0 || remoteEtag[0] == "" {
		return "", false
	}
	// When we get etags, they appear to be quoted.
	etag := strings.Trim(remoteEtag[0], `"`)
	return etag, etag != ""
}

type cachePlacer func(*http.Response) (string, error)

func (t *cacheTransport) retrieveAndSaveFile(request *http.Request, cp cachePlacer) (*http.Response, error) {
	if t.wrapped == nil {
		return nil, fmt.Errorf("wrapped client is nil")
	}
	resp, err := t.wrapped.Do(request)
	if err != nil || resp.StatusCode != 200 {
		return resp, err
	}

	// Determine the file we will caching stuff in based on the URL/response
	cacheFile, err := cp(resp)
	if err != nil {
		return nil, err
	}
	cacheDir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("unable to create cache directory: %w", err)
	}

	// Stream the request response to a temporary file within the final cache
	// directory
	tmp, err := os.CreateTemp(cacheDir, "*.tmp")
	if err != nil {
		return nil, fmt.Errorf("unable to create a temporary cache file: %w", err)
	}
	if err := func() error {
		defer tmp.Close()
		if _, err := io.Copy(tmp, resp.Body); err != nil {
			return fmt.Errorf("unable to write to cache file: %w", err)
		}
		return nil
	}(); err != nil {
		return nil, err
	}

	// Now that we have the file has been written, rename to atomically populate
	// the cache
	if err := os.Rename(tmp.Name(), cacheFile); err != nil {
		return nil, fmt.Errorf("unable to populate cache: %v", err)
	}

	// return a handle to our file
	f2, err := os.Open(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open cache file: %w", err)
	}
	resp.Body = f2
	return resp, nil
}

func cacheDirForPackage(root string, pkg *repository.RepositoryPackage) (string, error) {
	u, err := packageAsURL(pkg)
	if err != nil {
		return "", err
	}

	p, err := cachePathFromURL(root, *u)
	if err != nil {
		return "", err
	}

	if ext := filepath.Ext(p); ext != ".apk" {
		return "", fmt.Errorf("unexpected ext (%s) to cache dir: %q", ext, p)
	}

	return strings.TrimSuffix(p, ".apk"), nil
}

// cachePathFromURL given a URL, figure out what the cache path would be
func cachePathFromURL(root string, u url.URL) (string, error) {
	// the last two levels are what we append. For example https://example.com/foo/bar/x86_64/baz.apk
	// means we want to append x86_64/baz.apk to our cache root
	u2 := u
	u2.ForceQuery = false
	u2.RawFragment = ""
	u2.RawQuery = ""
	filename := filepath.Base(u2.Path)
	archDir := filepath.Dir(u2.Path)
	dir := filepath.Base(archDir)
	repoDir := filepath.Dir(archDir)
	// include the hostname
	u2.Path = repoDir

	// url encode it so it can be a single directory
	repoDir = url.QueryEscape(u2.String())
	cacheFile := filepath.Join(root, repoDir, dir, filename)
	// validate it is within root
	cacheFile = filepath.Clean(cacheFile)
	cleanroot := filepath.Clean(root)
	if !strings.HasPrefix(cacheFile, cleanroot) {
		return "", fmt.Errorf("cache file %s is not within root %s", cacheFile, cleanroot)
	}
	return cacheFile, nil
}
