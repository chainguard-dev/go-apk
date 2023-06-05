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
)

// cache
type cache struct {
	dir string
}

// client return an http.Client that knows how to read from and write to the cache
// key is in the implementation of https://pkg.go.dev/net/http#RoundTripper
func (c cache) client(wrapped *http.Client, etagRequired bool) *http.Client {
	return &http.Client{
		Transport: &cacheTransport{
			wrapped:      wrapped,
			root:         c.dir,
			etagRequired: etagRequired,
		},
	}
}

type cacheTransport struct {
	wrapped      *http.Client
	root         string
	etagRequired bool
}

func (t *cacheTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// do we have the file in the cache?
	if request.URL == nil {
		return nil, fmt.Errorf("no URL in request")
	}
	cacheFile, err := t.cachePathFromURL(*request.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid cache path based on URL: %w", err)
	}
	etagFile := etagFilename(cacheFile)
	// see if the file is in the cachel if not, just return the wrapped client call
	if _, err := os.Stat(cacheFile); err != nil {
		return t.retrieveAndSaveFile(cacheFile, etagFile, request)
	}
	// we found the file, see if we can get an etag for it
	// if no local etag, it means we are fine with the file itself without checking if it changed upstream
	etag, err := os.ReadFile(etagFile)
	if err != nil {
		if t.etagRequired {
			return t.retrieveAndSaveFile(cacheFile, etagFile, request)
		}
		// no etag, just return the file
		f, err := os.Open(cacheFile)
		if err != nil {
			return &http.Response{StatusCode: 404}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       f,
		}, nil
	}
	resp, err := t.wrapped.Head(request.URL.String())
	if err != nil || resp.StatusCode != 200 {
		return resp, err
	}
	remoteEtag, ok := resp.Header[http.CanonicalHeaderKey("etag")]
	if !ok || len(remoteEtag) == 0 || remoteEtag[0] == "" {
		return t.retrieveAndSaveFile(cacheFile, etagFile, request)
	}
	// did not match, our file is out of date, replace
	if string(etag) != remoteEtag[0] {
		_ = os.Remove(cacheFile)
		_ = os.Remove(etagFile)
		return t.retrieveAndSaveFile(cacheFile, etagFile, request)
	}
	// it matched, so use our cache file
	f, err := os.Open(cacheFile)
	if err != nil {
		return &http.Response{StatusCode: 404}, nil
	}
	return &http.Response{
		StatusCode: 200,
		Body:       f,
	}, nil
}

func (t *cacheTransport) retrieveAndSaveFile(cacheFile, etagFile string, request *http.Request) (*http.Response, error) {
	if t.wrapped == nil {
		return nil, fmt.Errorf("wrapped client is nil")
	}
	resp, err := t.wrapped.Do(request)
	if err != nil || resp.StatusCode != 200 {
		return resp, err
	}
	// save the file
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		return nil, fmt.Errorf("unable to create cache directory: %w", err)
	}
	f, err := os.Create(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("unable to create cache file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return nil, fmt.Errorf("unable to write to cache file: %w", err)
	}
	// was there an etag?
	etag := resp.Header.Get("etag")
	if etag != "" {
		if err := os.WriteFile(etagFile, []byte(etag), 0644); err != nil { //nolint:gosec // is ok for this file to be world-readable
			return nil, fmt.Errorf("unable to write etag file: %w", err)
		}
	}
	// return a handler to our file
	f2, err := os.Open(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open cache file: %w", err)
	}
	resp.Body = f2
	return resp, nil
}

// cachePathFromURL given a URL, figure out what the cache path would be
func (t *cacheTransport) cachePathFromURL(u url.URL) (string, error) {
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
	cacheFile := filepath.Join(t.root, repoDir, dir, filename)
	// validate it is within t.root
	cacheFile = filepath.Clean(cacheFile)
	root := filepath.Clean(t.root)
	if !strings.HasPrefix(cacheFile, root) {
		return "", fmt.Errorf("cache file %s is not within root %s", cacheFile, root)
	}
	return cacheFile, nil
}

func etagFilename(p string) string {
	return p + ".etag"
}
