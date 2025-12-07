package memfs

import (
	"context"
	"io"
	"time"

	"lesiw.io/fs"
)

var _ fs.CreateFS = (*memFS)(nil)

func (f *memFS) Create(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	name = resolvePath(ctx, name)
	f.Lock()
	defer f.Unlock()

	dir, name, ok := f.walkDir(name)
	if !ok {
		return nil, &fs.PathError{
			Op: "create", Path: name, Err: fs.ErrNotExist,
		}
	}

	n, ok := dir.nodes[name]
	if !ok {
		n = &node{
			name:    name,
			mode:    fs.FileMode(ctx),
			modTime: time.Now(),
		}
		dir.nodes[name] = n
	}
	n.data = nil
	return newWriter(f, n), nil
}
