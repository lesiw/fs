package memfs

import (
	"context"
	"io"
	"time"

	"lesiw.io/fs"
)

var _ fs.AppendFS = (*memFS)(nil)

func (f *memFS) Append(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	name = resolvePath(ctx, name)
	f.Lock()
	defer f.Unlock()

	dir, name, ok := f.walkDir(name)
	if !ok {
		return nil, &fs.PathError{
			Op: "append", Path: name, Err: fs.ErrNotExist,
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
	if n.dir {
		return nil, &fs.PathError{Op: "append", Path: name, Err: errIsDir}
	}
	return newWriter(f, n), nil
}
