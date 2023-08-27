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

package tarfs

import (
	"archive/tar"
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sync"
)

type Entry struct {
	tar.Header
	Offset int64
}

type File struct {
	fsys   *FS
	handle io.ReadSeekCloser
	r      io.Reader
	Entry  Entry
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.Entry.FileInfo(), nil
}

func (f *File) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func (f *File) Close() error {
	f.fsys.readers.Put(f.handle)
	return nil
}

type FS struct {
	readers sync.Pool
	files   []Entry
	index   map[string]int
}

// Open implements fs.FS.
func (fsys *FS) Open(name string) (fs.File, error) {
	i, ok := fsys.index[name]
	if !ok {
		return nil, fs.ErrNotExist
	}

	e := fsys.files[i]

	f := &File{
		fsys:  fsys,
		Entry: e,
	}

	if e.Size != 0 {
		rc, err := fsys.OpenAt(e.Offset)
		if err != nil {
			return nil, err
		}
		f.handle = rc
		f.r = io.LimitReader(rc, e.Size)
	}

	return f, nil
}

func (fsys *FS) Entries() []Entry {
	return fsys.files
}

func (fsys *FS) OpenAt(offset int64) (io.ReadSeekCloser, error) {
	v := fsys.readers.Get()
	if err, ok := v.(error); ok {
		return nil, err
	}
	if rsc, ok := v.(io.ReadSeekCloser); ok {
		if _, err := rsc.Seek(offset, io.SeekStart); err != nil {
			return nil, err
		}
		return rsc, nil
	}

	return nil, fmt.Errorf("unexpected type: %T", v)
}

type countReader struct {
	r io.Reader
	n int64
}

func (cr *countReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	cr.n += int64(n)
	return n, err
}

func New(open func() (io.ReadSeekCloser, error)) (*FS, error) {
	fsys := &FS{
		readers: sync.Pool{
			New: func() any {
				r, err := open()
				if err != nil {
					return err
				}
				return r
			},
		},
		files: []Entry{},
		index: map[string]int{},
	}

	// TODO: Consider caching this across builds.
	r, err := open()
	if err != nil {
		return nil, err
	}

	cr := &countReader{bufio.NewReaderSize(r, 1<<20), 0}
	tr := tar.NewReader(cr)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		fsys.index[hdr.Name] = len(fsys.files)
		fsys.files = append(fsys.files, Entry{
			Header: *hdr,
			Offset: cr.n,
		})
	}

	fsys.readers.Put(r)

	return fsys, nil
}
