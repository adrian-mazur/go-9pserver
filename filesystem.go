package main

import (
	"errors"
	"io"
	"log"
	"os"
	p "path"
	"strings"
	"sync"
)

type Filesystem interface {
	Open(path string, mode uint8) (File, error)
	CreateDir(path string) error
	CreateFile(path string) error
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
	Write(offset uint64, data []byte) error
	Close()
}

var ErrDoesNotExist = errors.New("no such file or directory")
var ErrIOError = errors.New("i/o error")
var ErrAlreadyExists = errors.New("file or directory already exists")
var ErrDirectoryNotEmpty = errors.New("directory not empty")

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

func (f *localFilesystem) Open(path string, mode uint8) (File, error) {
	fullPath := f.normalizePath(path)
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrDoesNotExist
		}
		log.Println(err)
		return nil, ErrIOError
	}
	if fileInfo.IsDir() {
		return &localFile{nil, fileInfo, f.qidPath(path), path == "/"}, nil
	}
	modeToFlag := map[uint8]int{OREAD: os.O_RDONLY, OWRITE: os.O_WRONLY, ORDWR: os.O_RDWR}
	flag := modeToFlag[mode|ORDWR]
	if mode&OTRUNC != 0 {
		flag |= os.O_TRUNC
	}
	file, err := os.OpenFile(fullPath, flag, os.ModePerm)
	if err != nil {
		log.Println(err)
		return nil, ErrIOError
	}
	return &localFile{file, fileInfo, f.qidPath(path), path == "/"}, nil
}

func (f *localFilesystem) CreateDir(path string) error {
	fullPath := f.normalizePath(path)
	if _, err := os.Stat(fullPath); !errors.Is(err, os.ErrNotExist) {
		return ErrAlreadyExists
	}
	err := os.Mkdir(fullPath, os.ModePerm)
	if err != nil {
		log.Println(err)
		return ErrIOError
	}
	return nil
}

func (f *localFilesystem) CreateFile(path string) error {
	fullPath := f.normalizePath(path)
	if _, err := os.Stat(fullPath); !errors.Is(err, os.ErrNotExist) {
		return ErrAlreadyExists
	}
	file, err := os.Create(fullPath)
	if err != nil {
		log.Println(err)
		return ErrIOError
	}
	_ = file.Close()
	return nil
}

func (f *localFilesystem) ReadDir(path string) ([]Stat, error) {
	entries, err := os.ReadDir(f.normalizePath(path))
	if err != nil {
		log.Println(err)
		return nil, ErrIOError
	}
	stats := make([]Stat, len(entries))
	for i, entry := range entries {
		fileInfo, err := entry.Info()
		if err != nil {
			log.Println(err)
			return nil, ErrIOError
		}
		qid := Qid{qidFtype(fileInfo.IsDir()), uint32(fileInfo.ModTime().Unix()), f.qidPath(p.Join(path, fileInfo.Name()))}
		var length uint64
		if fileInfo.IsDir() {
			length = 0
		} else {
			length = uint64(fileInfo.Size())
		}
		stats[i] = Stat{
			Qid:    qid,
			Mode:   0755 | (uint32(qid.Ftype) << 24),
			Length: length,
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

func (f *localFilesystem) Remove(path string) error {
	fullPath := f.normalizePath(path)
	err := os.Remove(fullPath)
	if err != nil {
		if strings.Contains(err.Error(), "not empty") {
			return ErrDirectoryNotEmpty
		}
		log.Println(err)
		return ErrIOError
	}
	return err
}

func (f *localFilesystem) Stat(path string) (Stat, error) {
	file, err := f.Open(f.normalizePath(path), OREAD)
	if err != nil {
		log.Println(err)
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
		name = f.osFileInfo.Name()
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
		log.Println(err)
		return nil, ErrIOError
	}
	return buffer[:n], nil
}

func (f *localFile) Write(offset uint64, data []byte) error {
	_, err := f.osFile.WriteAt(data, int64(offset))
	if err != nil {
		log.Println(err)
		return ErrIOError
	}
	return nil
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
