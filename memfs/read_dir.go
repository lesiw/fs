package memfs

import (
	"context"
	"iter"

	"lesiw.io/fs"
)

var _ fs.ReadDirFS = (*memFS)(nil)

func (f *memFS) ReadDir(
	ctx context.Context, name string,
) iter.Seq2[fs.DirEntry, error] {
	name = resolvePath(ctx, name)
	if name == "." || name == "/" {
		name = ""
	}

	return func(yield func(fs.DirEntry, error) bool) {
		// Snapshot entries while holding lock
		f.RLock()

		n, ok := f.walk(name)
		if !ok {
			f.RUnlock()
			yield(nil, &fs.PathError{
				Op: "readdir", Path: name, Err: fs.ErrNotExist,
			})
			return
		}
		if !n.dir {
			f.RUnlock()
			yield(nil, &fs.PathError{
				Op: "readdir", Path: name, Err: fs.ErrNotDir,
			})
			return
		}

		// Snapshot children
		var entries []*dirEntry
		for _, child := range n.nodes {
			entries = append(entries, &dirEntry{
				name:  child.name,
				isDir: child.dir,
				typ:   child.mode.Type(),
				info:  &fileInfo{node: child},
			})
		}
		f.RUnlock()

		// Yield entries without holding lock
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}
			if !yield(entry, nil) {
				return
			}
		}
	}
}

// dirEntry implements fs.DirEntry.
type dirEntry struct {
	name  string
	isDir bool
	typ   fs.Mode
	info  fs.FileInfo
}

func (de *dirEntry) Name() string               { return de.name }
func (de *dirEntry) IsDir() bool                { return de.isDir }
func (de *dirEntry) Type() fs.Mode              { return de.typ }
func (de *dirEntry) Info() (fs.FileInfo, error) { return de.info, nil }
func (de *dirEntry) Path() string               { return "" }
