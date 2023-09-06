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
	"io"
	"io/fs"
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
	return f.handle.Close()
}

type FS struct {
	open  func() (io.ReadSeekCloser, error)
	files []Entry
	index map[string]int
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
	// TODO: We can use ReadAt to avoid opening the file multiple times.
	f, err := fsys.open()
	if err != nil {
		return nil, err
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	return f, nil
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
		open:  open,
		files: []Entry{},
		index: map[string]int{},
	}

	// TODO: Consider caching this across builds.
	r, err := open()
	if err != nil {
		return nil, err
	}
	defer r.Close()

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

	return fsys, nil
}
