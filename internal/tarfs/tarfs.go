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
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"time"

	"golang.org/x/exp/slices"
)

type Entry struct {
	Header tar.Header
	Offset int64

	dir string
	fi  fs.FileInfo
}

func (e Entry) Name() string {
	return e.fi.Name()
}

func (e Entry) Size() int64 {
	return e.Header.Size
}

func (e Entry) Type() fs.FileMode {
	return e.fi.Mode()
}

func (e Entry) Info() (fs.FileInfo, error) {
	return e.fi, nil
}

func (e Entry) IsDir() bool {
	return e.fi.IsDir()
}

type File struct {
	fsys   *FS
	handle io.ReadSeekCloser
	r      io.Reader
	Entry  *Entry
}

func (f *File) Stat() (fs.FileInfo, error) {
	return f.Entry.fi, nil
}

func (f *File) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func (f *File) relativeOffset(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		return offset, nil
	case io.SeekStart:
		if offset > f.Entry.Header.Size {
			return 0, fmt.Errorf("offset %d greater than file size %d", offset, f.Entry.Header.Size)
		}
	case io.SeekEnd:
		if offset+f.Entry.Header.Size < 0 {
			return 0, fmt.Errorf("offset %d greater than file size %d", offset, f.Entry.Header.Size)
		}
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	return f.Entry.Offset + offset, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	newOffset, err := f.relativeOffset(offset, whence)
	if err != nil {
		return 0, err
	}

	n, err := f.handle.Seek(newOffset, whence)
	if err != nil {
		return 0, err
	}

	return n - f.Entry.Offset, nil
}

func (f *File) ReadAt(p []byte, off int64) (int, error) {
	// If the underlying ReadSeekCloser implements ReaderAt, just use that.
	if ra, ok := f.handle.(io.ReaderAt); ok {
		bytesLeft := f.Entry.Header.Size - off
		if int64(len(p)) > bytesLeft {
			p = p[:bytesLeft]
		}
		return ra.ReadAt(p, f.Entry.Offset+off)
	}

	// Otherwise do a Seek and ReadFull.
	if _, err := f.handle.Seek(f.Entry.Offset+off, io.SeekStart); err != nil {
		return 0, err
	}
	f.r = io.LimitReader(f.handle, f.Entry.Header.Size-off)

	return io.ReadFull(f.r, p)
}

func (f *File) Close() error {
	return f.handle.Close()
}

type FS struct {
	open  func() (io.ReadSeekCloser, error)
	files []*Entry
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

	if e.Header.Size == 0 {
		return f, nil
	}

	rc, err := fsys.OpenAt(e.Offset)
	if err != nil {
		return nil, err
	}
	f.handle = rc
	f.r = io.LimitReader(rc, e.Header.Size)

	return f, nil
}

func (fsys *FS) Entries() []*Entry {
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

type root struct{}

func (r root) Name() string       { return "." }
func (r root) Size() int64        { return 0 }
func (r root) Mode() fs.FileMode  { return fs.ModeDir }
func (r root) ModTime() time.Time { return time.Unix(0, 0) }
func (r root) IsDir() bool        { return true }
func (r root) Sys() any           { return nil }

func (fsys *FS) Stat(name string) (fs.FileInfo, error) {
	if i, ok := fsys.index[name]; ok {
		return fsys.files[i].fi, nil
	}

	// fs.WalkDir expects "." to return a root entry to bootstrap the walk.
	// If we didn't find it above, synthesize one.
	if name == "." {
		return root{}, nil
	}

	return nil, fs.ErrNotExist
}

func (fsys *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	children := []fs.DirEntry{}
	for _, f := range fsys.files {
		// This is load bearing.
		f := f

		if f.dir != name {
			continue
		}

		children = append(children, f)
	}

	slices.SortFunc(children, func(a, b fs.DirEntry) int {
		return cmp.Compare(a.Name(), b.Name())
	})

	return children, nil
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
		files: []*Entry{},
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
		fsys.files = append(fsys.files, &Entry{
			Header: *hdr,
			Offset: cr.n,
			dir:    path.Dir(hdr.Name),
			fi:     hdr.FileInfo(),
		})
	}

	return fsys, nil
}
