package s3

import (
	"time"

	"lesiw.io/fs"
)

// s3FileInfo implements fs.FileInfo for S3 objects
type s3FileInfo struct {
	name string
	size int64
	mode fs.Mode
	time time.Time
}

func (fi *s3FileInfo) Name() string       { return fi.name }
func (fi *s3FileInfo) Size() int64        { return fi.size }
func (fi *s3FileInfo) Mode() fs.Mode      { return fi.mode }
func (fi *s3FileInfo) ModTime() time.Time { return fi.time }
func (fi *s3FileInfo) Sys() any           { return nil }

func (fi *s3FileInfo) IsDir() bool { return fi.mode.IsDir() }

// s3DirEntry implements fs.DirEntry for S3 objects
type s3DirEntry struct {
	name  string
	isDir bool
	size  int64
	time  time.Time
}

func (de *s3DirEntry) Name() string { return de.name }
func (de *s3DirEntry) IsDir() bool  { return de.isDir }
func (de *s3DirEntry) Type() fs.Mode {
	if de.isDir {
		return fs.ModeDir
	}
	return 0
}

func (de *s3DirEntry) Info() (fs.FileInfo, error) {
	mode := fs.Mode(0644)
	if de.isDir {
		mode = fs.ModeDir | 0755
	}
	return &s3FileInfo{
		name: de.name,
		size: de.size,
		mode: mode,
		time: de.time,
	}, nil
}

func (de *s3DirEntry) Path() string { return "" }
