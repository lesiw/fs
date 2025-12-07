package memfs

import (
	"context"
	"time"

	"lesiw.io/fs"
)

var _ fs.TruncateFS = (*memFS)(nil)

func (f *memFS) Truncate(ctx context.Context, name string, size int64) error {
	name = resolvePath(ctx, name)
	f.Lock()
	defer f.Unlock()

	n, ok := f.walk(name)
	if !ok {
		return &fs.PathError{Op: "truncate", Path: name, Err: fs.ErrNotExist}
	}
	if n.dir {
		return &fs.PathError{Op: "truncate", Path: name, Err: errIsDir}
	}

	if size < 0 {
		size = 0
	}

	if int64(len(n.data)) > size {
		n.data = n.data[:size]
	} else if int64(len(n.data)) < size {
		newData := make([]byte, size)
		copy(newData, n.data)
		n.data = newData
	}
	n.modTime = time.Now()

	return nil
}
