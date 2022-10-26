package server

import (
	"bytes"
	"compress/gzip"
	"io/fs"
	"strings"
	"time"
)

// fsDecomp can be used to wrap a fs.FS. If a file is requested and not found,
// we look for a .gz version. If the .gz version is found, we decompress it
// and return the contents. This allows us to ship .gz compressed embedded files
// but still serve uncompressed files.
type fsDecomp struct {
	fs fs.FS
}

func newFsDecomp(fs fs.FS) *fsDecomp {
	return &fsDecomp{fs: fs}
}

func (fsd *fsDecomp) Open(name string) (fs.File, error) {
	// look for file, if it does not exist, look for gz version
	f, err := fsd.fs.Open(name)
	if err != nil {
		f, gzerr := fsd.fs.Open(name + ".gz")
		if gzerr != nil {
			// return original error
			return f, err
		}

		// return fileGz version
		return newFileGz(f)
	}

	return f, nil
}

// fileGz implements both fs.File and fs.FileInfo interfaces
type fileGz struct {
	file     fs.File
	fileInfo fs.FileInfo
	data     bytes.Buffer
	size     int64
}

func newFileGz(file fs.File) (*fileGz, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	r, err := gzip.NewReader(file)

	if err != nil {
		return nil, err
	}

	var data bytes.Buffer

	size, err := data.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	return &fileGz{file: file, fileInfo: fileInfo, data: data, size: size}, nil
}

func (fgz *fileGz) Stat() (fs.FileInfo, error) {
	return fgz, nil
}

func (fgz *fileGz) Read(data []byte) (int, error) {
	return fgz.data.Read(data)
}

func (fgz *fileGz) Close() error {
	return nil
}

func (fgz *fileGz) Name() string {
	return strings.TrimSuffix(fgz.fileInfo.Name(), ".gz")
}

func (fgz *fileGz) Size() int64 {
	return fgz.size
}

func (fgz *fileGz) Mode() fs.FileMode {
	return fgz.fileInfo.Mode()
}

func (fgz *fileGz) ModTime() time.Time {
	return fgz.fileInfo.ModTime()
}

func (fgz *fileGz) IsDir() bool {
	return fgz.fileInfo.IsDir()
}

func (fgz *fileGz) Sys() any {
	return fgz.fileInfo.Sys()
}
