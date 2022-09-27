package main

import (
	"errors"
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
