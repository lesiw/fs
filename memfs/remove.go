package memfs

import (
	"context"

	"lesiw.io/fs"
)

var _ fs.RemoveFS = (*memFS)(nil)

func (f *memFS) Remove(ctx context.Context, name string) error {
	name = resolvePath(ctx, name)
	f.Lock()
	defer f.Unlock()

	dir, name, ok := f.walkDir(name)
	if !ok {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}

	n, ok := dir.nodes[name]
	if !ok {
		return &fs.PathError{Op: "remove", Path: name, Err: fs.ErrNotExist}
	}

	if n.dir && len(n.nodes) > 0 {
		return &fs.PathError{
			Op: "remove", Path: name, Err: errDirNotEmpty,
		}
	}

	delete(dir.nodes, name)
	return nil
}
