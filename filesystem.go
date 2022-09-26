package main

import (
	"errors"
	"io"
	"os"
	p "path"
	"sync"
)

type Filesystem interface {
	Open(path string) (File, error)
	Create(path string) error
	Write(file File, offset uint64, count uint32, data []byte) error
	ReadDir(path string) ([]Stat, error)
	Remove(path string) error
	Stat(path string) (Stat, error)
	Wstat(path string, stat Stat) error
}

type File interface {
	Qid() Qid
	IsDir() bool
	Stat() (Stat, error)
	Read(offset uint64, count uint32) ([]byte, error)
	Close()
}

var ErrDoesNotExist = errors.New("no such file or directory")
var ErrIOError = errors.New("i/o error")

type localFilesystem struct {
	basePath string

	qidMutex   sync.Mutex
	qidCounter uint64
	qidMap     map[string]uint64
}

type localFile struct {
	osFile     *os.File
	osFileInfo os.FileInfo
	qidPath    uint64
	isRoot     bool
}

func NewLocalFilesystem(basePath string) Filesystem {
	var l localFilesystem
	l.basePath = basePath
	l.qidMap = make(map[string]uint64)
	return &l
}

func (f *localFilesystem) Open(path string) (File, error) {
	file, err := os.Open(f.normalizePath(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrDoesNotExist
		}
		return nil, ErrIOError
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, ErrIOError
	}
	if fileInfo.IsDir() {
		_ = file.Close()
	}
	return &localFile{file, fileInfo, f.qidPath(path), path == "/"}, nil
}

func (f *localFilesystem) Create(path string) error { // TODO
	return nil
}

func (f *localFilesystem) Write(file File, offset uint64, count uint32, data []byte) error { // TODO
	return nil
}

func (f *localFilesystem) ReadDir(path string) ([]Stat, error) {
	entries, err := os.ReadDir(f.normalizePath(path))
	if err != nil {
		return nil, ErrIOError
	}
	stats := make([]Stat, len(entries))
	for i, entry := range entries {
		fileInfo, err := entry.Info()
		if err != nil {
			return nil, ErrIOError
		}
		qid := Qid{qidFtype(fileInfo.IsDir()), uint32(fileInfo.ModTime().Unix()), f.qidPath(p.Join(path, fileInfo.Name()))}
		stats[i] = Stat{
			Qid:    qid,
			Mode:   0755 | (uint32(qid.Ftype) << 24),
			Length: uint64(fileInfo.Size()),
			Name:   fileInfo.Name(),
			Uid:    "?",
			Gid:    "?",
			Muid:   "",
			Atime:  uint32(fileInfo.ModTime().Unix()),
			Mtime:  uint32(fileInfo.ModTime().Unix()),
		}
	}
	return stats, nil
}

func (f *localFilesystem) Remove(path string) error { // TODO
	return nil
}

func (f *localFilesystem) Stat(path string) (Stat, error) {
	file, err := f.Open(path)
	if err != nil {
		return Stat{}, err
	}
	defer file.Close()
	return file.Stat()
}

func (f *localFilesystem) Wstat(path string, stat Stat) error { // TODO
	return nil
}

func (f *localFilesystem) normalizePath(path string) string {
	return p.Join(f.basePath, p.Clean(path))
}

func (f *localFilesystem) qidPath(path string) uint64 {
	f.qidMutex.Lock()
	defer f.qidMutex.Unlock()
	qidPath, ok := f.qidMap[path]
	if ok {
		return qidPath
	}
	f.qidMap[path] = f.qidCounter
	f.qidCounter += 1
	return f.qidMap[path]
}

func (f *localFile) Qid() Qid {
	return Qid{qidFtype(f.IsDir()), uint32(f.osFileInfo.ModTime().Unix()), f.qidPath}
}

func (f *localFile) IsDir() bool {
	return f.osFileInfo.IsDir()
}

func (f *localFile) Stat() (Stat, error) {
	var name string
	if f.isRoot {
		name = "/"
	} else {
		name = f.osFile.Name()
	}
	return Stat{
		Qid:    f.Qid(),
		Mode:   0755 | (uint32(f.Qid().Ftype) << 24),
		Length: uint64(f.osFileInfo.Size()),
		Name:   name,
		Uid:    "?",
		Gid:    "?",
		Muid:   "",
		Atime:  uint32(f.osFileInfo.ModTime().Unix()),
		Mtime:  uint32(f.osFileInfo.ModTime().Unix()),
	}, nil
}

func (f *localFile) Read(offset uint64, count uint32) ([]byte, error) {
	buffer := make([]byte, count)
	n, err := f.osFile.ReadAt(buffer, int64(offset))
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, ErrIOError
	}
	return buffer[:n], nil
}

func (f *localFile) Close() {
	if !f.IsDir() {
		_ = f.osFile.Close()
	}
}

func qidFtype(isDir bool) uint8 {
	if isDir {
		return DMDIR >> 24
	} else {
		return 0
	}
}
