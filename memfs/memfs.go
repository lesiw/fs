// Package memfs implements lesiw.io/fs.FS using an in-memory file tree.
package memfs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"lesiw.io/fs"
)

var (
	errIsDir       = errors.New("is a directory")
	errDirNotEmpty = errors.New("directory not empty")
)

// New returns a new empty in-memory filesystem.
func New() fs.FS {
	return &memFS{
		node: &node{
			name:    "",
			mode:    0755 | fs.ModeDir,
			modTime: time.Now(),
			dir:     true,
			nodes:   make(map[string]*node),
		},
	}
}

type memFS struct {
	sync.RWMutex
	*node
}

// node represents a file or directory in the filesystem.
type node struct {
	name    string
	data    []byte
	mode    fs.Mode
	modTime time.Time
	dir     bool
	nodes   map[string]*node
}

// resolvePath resolves a path relative to WorkDir if present.
func resolvePath(ctx context.Context, name string) string {
	name = path.Clean(name)
	if workDir := fs.WorkDir(ctx); workDir != "" {
		name = path.Join(workDir, name)
	}
	return name
}

// walk traverses the tree to find a node at the given path.
// Returns the node and true if found, nil and false otherwise.
func (f *memFS) walk(name string) (*node, bool) {
	if name == "." || name == "" || name == "/" {
		return f.node, true
	}

	parts := strings.Split(name, "/")
	current := f.node

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if !current.dir {
			return nil, false
		}
		next, ok := current.nodes[part]
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func (f *memFS) walkDir(name string) (*node, string, bool) {
	dir, base := path.Split(name)
	parent, ok := f.walk(dir)
	if !ok || !parent.dir {
		return nil, name, false
	}
	return parent, base, true
}

var _ fs.FS = (*memFS)(nil)

func (f *memFS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	name = resolvePath(ctx, name)
	f.RLock()
	defer f.RUnlock()

	n, ok := f.walk(name)
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	if n.dir {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errIsDir}
	}

	return io.NopCloser(bytes.NewReader(n.data)), nil
}

type writer struct {
	*memFS
	*node
	bytes.Buffer
}

func newWriter(fs *memFS, n *node) *writer {
	return &writer{memFS: fs, node: n}
}

func (w *writer) Write(p []byte) (int, error) { return w.Buffer.Write(p) }

func (w *writer) Close() error {
	w.Lock()
	defer w.Unlock()

	w.node.data = append(w.node.data, w.Bytes()...)
	w.node.modTime = time.Now()

	return nil
}
