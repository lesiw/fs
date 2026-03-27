// Package memfs implements lesiw.io/fs.FS using an in-memory file tree.
package memfs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
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
	target  string // symlink target
}

// resolvePath resolves a path relative to WorkDir if present.
func resolvePath(ctx context.Context, name string) string {
	name = path.Clean(name)
	if w := fs.WorkDir(ctx); w != "" && !path.IsAbs(name) {
		name = path.Join(w, name)
	}
	return name
}

// walk traverses the tree to find a node at the given path,
// following symlinks.
func (f *memFS) walk(name string) (*node, bool) {
	return f.resolve(name, true, 0)
}

// walkNoFollow is like walk but does not follow the final symlink.
func (f *memFS) walkNoFollow(name string) (*node, bool) {
	return f.resolve(name, false, 0)
}

// resolve traverses the tree to find a node at the given path.
// If follow is true, symlinks at the final component are followed.
// Symlinks in intermediate components are always followed.
func (f *memFS) resolve(name string, follow bool, depth int) (*node, bool) {
	if depth > 255 {
		return nil, false
	}
	if name == "." || name == "" || name == "/" {
		return f.node, true
	}

	parts := strings.Split(name, "/")
	current := f.node

	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if !current.dir {
			return nil, false
		}
		child, ok := current.nodes[part]
		if !ok {
			return nil, false
		}
		last := i == len(parts)-1
		if child.target != "" && (follow || !last) {
			target := child.target
			if !path.IsAbs(target) {
				parent := path.Join(parts[:i]...)
				target = path.Join(parent, target)
			}
			if !last {
				remaining := path.Join(parts[i+1:]...)
				target = path.Join(target, remaining)
			}
			return f.resolve(target, follow, depth+1)
		}
		current = child
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

func (w *writer) Write(p []byte) (int, error) {
	w.Lock()
	defer w.Unlock()
	return w.Buffer.Write(p)
}

func (w *writer) Close() error {
	w.Lock()
	defer w.Unlock()

	w.node.data = append(w.node.data, w.Bytes()...)
	w.node.modTime = time.Now()

	return nil
}
