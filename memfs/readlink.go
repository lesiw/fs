package memfs

import (
	"context"

	"lesiw.io/fs"
)

var _ fs.ReadLinkFS = (*memFS)(nil)

func (f *memFS) ReadLink(ctx context.Context, name string) (string, error) {
	name = resolvePath(ctx, name)
	f.RLock()
	defer f.RUnlock()

	n, ok := f.walkNoFollow(name)
	if !ok {
		return "", &fs.PathError{
			Op: "readlink", Path: name, Err: fs.ErrNotExist,
		}
	}
	if n.target == "" {
		return "", &fs.PathError{
			Op: "readlink", Path: name, Err: fs.ErrInvalid,
		}
	}
	return n.target, nil
}

func (f *memFS) Lstat(ctx context.Context, name string) (fs.FileInfo, error) {
	name = resolvePath(ctx, name)
	f.RLock()
	defer f.RUnlock()

	n, ok := f.walkNoFollow(name)
	if !ok {
		return nil, &fs.PathError{
			Op: "lstat", Path: name, Err: fs.ErrNotExist,
		}
	}
	return &fileInfo{node: n}, nil
}
