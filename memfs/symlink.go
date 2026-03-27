package memfs

import (
	"context"
	"time"

	"lesiw.io/fs"
)

var _ fs.SymlinkFS = (*memFS)(nil)

func (f *memFS) Symlink(ctx context.Context, oldname, newname string) error {
	newname = resolvePath(ctx, newname)
	f.Lock()
	defer f.Unlock()

	dir, base, ok := f.walkDir(newname)
	if !ok {
		return &fs.PathError{
			Op: "symlink", Path: newname, Err: fs.ErrNotExist,
		}
	}

	if _, exists := dir.nodes[base]; exists {
		return &fs.PathError{
			Op: "symlink", Path: newname, Err: fs.ErrExist,
		}
	}

	dir.nodes[base] = &node{
		name:    base,
		mode:    0777 | fs.ModeSymlink,
		modTime: time.Now(),
		target:  oldname,
	}
	return nil
}
