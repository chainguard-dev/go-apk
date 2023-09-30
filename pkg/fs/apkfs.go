package fs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/chainguard-dev/go-apk/pkg/expandapk"
)

type APKFS struct {
	path  string
	files map[string]*apkFSFile
	ctx   context.Context
	cache *expandapk.APKExpanded
}

func (a *APKFS) acquireCache() (*expandapk.APKExpanded, error) {
	if a.cache == nil {
		file, err := os.Open(a.path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		a.cache, err = expandapk.ExpandApk(a.ctx, file, "/tmp/")
		if err != nil {
			return nil, err
		}
	}
	return a.cache, nil
}
func (a *APKFS) getTarReader() (*os.File, *tar.Reader, error) {
	file, err := os.Open(a.cache.PackageFile)

	if err != nil {
		return nil, nil, err
	}
	gzipStream, err := gzip.NewReader(file)
	if err != nil {
		return nil, nil, err
	}
	tr := tar.NewReader(gzipStream)
	return file, tr, nil
}
func NewAPKFS(ctx context.Context, archive string) (*APKFS, error) {
	result := APKFS{archive, make(map[string]*apkFSFile), ctx, nil}

	file, err := os.Open(archive)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	apkExpanded, err := expandapk.ExpandApk(ctx, file, "")
	if err != nil {
		return nil, err
	}
	defer apkExpanded.Close()
	gzipFile, err := os.Open(apkExpanded.PackageFile)
	if err != nil {
		return nil, err
	}
	defer gzipFile.Close()
	gzipStream, err := gzip.NewReader(gzipFile)
	if err != nil {
		return nil, err
	}

	reader := tar.NewReader(gzipStream)
	for {
		header, err := reader.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		currentEntry := apkFSFile{mode: fs.FileMode(header.Mode), name: "/" + header.Name,
			uid: header.Uid, gid: header.Gid,
			size: uint64(header.Size), modTime: header.ModTime,
			createTime: header.ChangeTime,
			linkTarget: header.Linkname, isDir: header.Typeflag == tar.TypeDir,
			xattrs: make(map[string][]byte)}
		for k, v := range header.PAXRecords {
			// If this trend continues then it would be wise to move the
			// named constant for this into a place accessible from here
			attrname := strings.TrimPrefix(k, "SCHILY.xattr.")
			if len(attrname) != len(k) {
				currentEntry.xattrs[attrname] = []byte(v)
			}
		}
		result.files["/"+header.Name] = &currentEntry
	}
	result.cache, err = result.acquireCache()
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func (a *APKFS) Close() error {
	if a.cache == nil {
		return nil
	}
	return a.cache.Close()
}

type apkFSFile struct {
	mode       fs.FileMode
	uid, gid   int
	name       string
	size       uint64
	modTime    time.Time
	createTime time.Time
	linkTarget string
	linkCount  int
	xattrs     map[string][]byte
	isDir      bool
	fs         *APKFS
	// The following fields are not initialized in the copies held
	// by the apkfs object.
	fileDescriptor io.Closer
	tarReader      *tar.Reader
}

// Users of the api should not handle the copies referred to in the
// filesystem object.
func (a *apkFSFile) acquireCopy() *apkFSFile {
	return &apkFSFile{mode: a.mode, uid: a.uid, gid: a.gid, size: a.size,
		name: a.name, modTime: a.modTime, createTime: a.createTime, linkTarget: a.linkTarget,
		linkCount: a.linkCount, xattrs: a.xattrs, isDir: a.isDir, fs: a.fs,
		fileDescriptor: nil, tarReader: nil}
}
func (a *apkFSFile) seekTo(reader *tar.Reader) error {
	for {
		header, err := reader.Next()
		if err == os.ErrNotExist {
			break
		} else if err != nil {
			return err
		}
		if header.Name == a.name[1:] {
			return nil
		}
	}
	return os.ErrNotExist
}

func (a *apkFSFile) Read(b []byte) (int, error) {
	return a.tarReader.Read(b)
}
func (a *apkFSFile) Stat() (fs.FileInfo, error) {
	return &apkFSFileInfo{file: a, name: a.name}, nil
}
func (a *apkFSFile) Close() error {
	if a.fileDescriptor != nil {
		err := a.fileDescriptor.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *APKFS) Stat(path string) (fs.FileInfo, error) {
	file, ok := a.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &apkFSFileInfo{file: file, name: file.name[strings.LastIndex(file.name, "/"):]}, nil
}

func (a *APKFS) Open(path string) (fs.File, error) {
	file, ok := a.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	fileCopy := file.acquireCopy()
	var err error
	fileCopy.fileDescriptor, fileCopy.tarReader, err = a.getTarReader()
	if err != nil {
		return nil, err
	}
	err = fileCopy.seekTo(fileCopy.tarReader)
	if err != nil {
		return nil, err
	}
	return fileCopy, nil
}

type apkFSFileInfo struct {
	file *apkFSFile
	name string
}

func (a *apkFSFileInfo) Name() string {
	return a.file.name[strings.LastIndex(a.name, "/")+1:]
}
func (a *apkFSFileInfo) Size() int64 {
	return int64(a.file.size)
}
func (a *apkFSFileInfo) Mode() fs.FileMode {
	return a.file.mode
}
func (a *apkFSFileInfo) ModTime() time.Time {
	return a.file.modTime
}
func (a *apkFSFileInfo) IsDir() bool {
	return a.file.isDir
}
func (a *apkFSFileInfo) Sys() any {
	return &tar.Header{
		Mode: int64(a.file.mode),
		Uid:  a.file.uid,
		Gid:  a.file.gid,
	}
}
