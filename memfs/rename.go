package memfs

import (
	"context"

	"lesiw.io/fs"
)

var _ fs.RenameFS = (*memFS)(nil)

func (f *memFS) Rename(ctx context.Context, oldname, newname string) error {
	oldname, newname = resolvePath(ctx, oldname), resolvePath(ctx, newname)
	f.Lock()
	defer f.Unlock()

	oldDir, oldname, ok := f.walkDir(oldname)
	if !ok {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrNotExist}
	}

	n, ok := oldDir.nodes[oldname]
	if !ok {
		return &fs.PathError{Op: "rename", Path: oldname, Err: fs.ErrNotExist}
	}

	newDir, newname, ok := f.walkDir(newname)
	if !ok {
		return &fs.PathError{
			Op: "rename", Path: newname, Err: fs.ErrNotExist,
		}
	}

	n.name = newname
	newDir.nodes[newname] = n
	delete(oldDir.nodes, oldname)

	return nil
}
