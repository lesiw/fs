package memfs

import (
	"context"
	"time"

	"lesiw.io/fs"
)

var _ fs.StatFS = (*memFS)(nil)

func (f *memFS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	name = resolvePath(ctx, name)
	f.RLock()
	defer f.RUnlock()

	n, ok := f.walk(name)
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}

	return &fileInfo{node: n}, nil
}

var _ fs.FileInfo = (*fileInfo)(nil)

type fileInfo struct{ *node }

func (fi *fileInfo) Name() string       { return fi.node.name }
func (fi *fileInfo) Size() int64        { return int64(len(fi.node.data)) }
func (fi *fileInfo) Mode() fs.Mode      { return fi.node.mode }
func (fi *fileInfo) ModTime() time.Time { return fi.node.modTime }
func (fi *fileInfo) IsDir() bool        { return fi.node.dir }
func (fi *fileInfo) Sys() any           { return nil }
