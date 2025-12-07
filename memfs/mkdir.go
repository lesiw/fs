package memfs

import (
	"context"
	"time"

	"lesiw.io/fs"
)

var _ fs.MkdirFS = (*memFS)(nil)

func (f *memFS) Mkdir(ctx context.Context, name string) error {
	name = resolvePath(ctx, name)

	// Special case: "." always exists
	if name == "." {
		return &fs.PathError{Op: "mkdir", Path: name, Err: fs.ErrExist}
	}

	f.Lock()
	defer f.Unlock()

	dir, name, ok := f.walkDir(name)
	if !ok {
		return &fs.PathError{Op: "mkdir", Path: name, Err: fs.ErrNotExist}
	}

	if _, exists := dir.nodes[name]; exists {
		return &fs.PathError{Op: "mkdir", Path: name, Err: fs.ErrExist}
	}

	dir.nodes[name] = &node{
		name:    name,
		mode:    fs.DirMode(ctx) | fs.ModeDir,
		modTime: time.Now(),
		dir:     true,
		nodes:   make(map[string]*node),
	}

	return nil
}
