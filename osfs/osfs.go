// Package osfs implements lesiw.io/fs.FS using the os package.
//
// This package is primarily intended for examples and testing. It provides a
// simple filesystem implementation backed by the operating system's native
// filesystem, supporting all optional interfaces defined in lesiw.io/fs.
//
// # Context Handling
//
// The lesiw.io/fs package uses context.Context throughout its API to support
// cancelation and timeouts for remote filesystem operations (e.g., cloud
// storage, network filesystems). However, since osfs uses the local os
// package, context cancelation does not apply and all context parameters
// are ignored.
//
// This is acceptable for an example implementation, but production filesystem
// implementations that perform I/O over a network should respect context
// cancelation.
package osfs

import (
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"time"

	"lesiw.io/fs"
)

// FS implements lesiw.io/fs.FS using the OS filesystem.
// It supports all optional interfaces defined in lesiw.io/fs.
//
// FS also implements io.Closer. If the filesystem was created with an empty
// root (which creates a temporary directory), Close() will remove the
// temporary directory.
type FS struct {
	root      string
	cleanupFn func() error
}

// New creates a new OS filesystem rooted at the specified directory.
//
// If root is empty (""), a temporary directory is created and the filesystem
// is rooted there. Call Close() to remove the temporary directory when done.
//
// If root is ".", it uses the current working directory.
//
// All paths are resolved relative to this root.
//
// Returns an error if the current working directory cannot be determined
// when root is ".", or if a temporary directory cannot be created when root
// is empty.
func New(root string) (*FS, error) {
	var cleanupFn func() error

	if root == "" {
		// Create temporary directory
		var err error
		root, err = os.MkdirTemp("", "osfs-*")
		if err != nil {
			return nil, fmt.Errorf("creating temp directory: %w", err)
		}
		cleanupFn = func() error {
			return os.RemoveAll(root)
		}
	} else if root == "." {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	return &FS{root: root, cleanupFn: cleanupFn}, nil
}

// resolvePath converts a relative fs path to an absolute OS path.
// If ctx contains a working directory via fs.WorkDir(), paths are resolved
// relative to that working directory within the filesystem root.
func (f *FS) resolvePath(ctx context.Context, name string) (string, error) {
	name = filepath.Clean(name)
	if filepath.IsAbs(name) {
		return name, nil
	}
	base := f.root
	if workDir := fs.WorkDir(ctx); workDir != "" {
		base = filepath.Join(f.root, filepath.FromSlash(workDir))
	}
	return filepath.Join(base, filepath.FromSlash(name)), nil
}

// Open implements fs.FS
func (f *FS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

// Create implements fs.CreateFS
func (f *FS) Create(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	perm := fs.FileMode(ctx)
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
}

// Append implements fs.AppendFS
func (f *FS) Append(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	perm := fs.FileMode(ctx)
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, perm)
}

// Stat implements fs.StatFS
func (f *FS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Stat(path)
}

// ReadDir implements fs.ReadDirFS
func (f *FS) ReadDir(
	ctx context.Context, name string,
) iter.Seq2[fs.DirEntry, error] {
	return func(yield func(fs.DirEntry, error) bool) {
		path, err := f.resolvePath(ctx, name)
		if err != nil {
			yield(nil, err)
			return
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			yield(nil, readErr)
			return
		}
		for _, entry := range entries {
			// Wrap os.DirEntry to include Path()/Depth() methods
			info, infoErr := entry.Info()
			if infoErr != nil {
				yield(nil, infoErr)
				return
			}
			wrapped := &dirEntry{
				name:  entry.Name(),
				isDir: entry.IsDir(),
				typ:   fs.Mode(entry.Type()),
				info:  info,
			}
			if !yield(wrapped, nil) {
				return
			}
		}
	}
}

// dirEntry implements fs.DirEntry without path/depth (for ReadDir).
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

// Remove implements fs.RemoveFS
func (f *FS) Remove(ctx context.Context, name string) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

// Mkdir implements fs.MkdirFS
func (f *FS) Mkdir(ctx context.Context, name string) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	perm := fs.DirMode(ctx)
	return os.Mkdir(path, perm)
}

// Rename implements fs.RenameFS
func (f *FS) Rename(ctx context.Context, oldname, newname string) error {
	oldpath, err := f.resolvePath(ctx, oldname)
	if err != nil {
		return err
	}
	newpath, err := f.resolvePath(ctx, newname)
	if err != nil {
		return err
	}
	return os.Rename(oldpath, newpath)
}

// Truncate implements fs.TruncateFS
func (f *FS) Truncate(ctx context.Context, name string, size int64) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Truncate(path, size)
}

// Chtimes implements fs.ChtimesFS
func (f *FS) Chtimes(
	ctx context.Context, name string, atime, mtime time.Time,
) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Chtimes(path, atime, mtime)
}

// Symlink implements fs.SymlinkFS
func (f *FS) Symlink(ctx context.Context, oldname, newname string) error {
	newpath, err := f.resolvePath(ctx, newname)
	if err != nil {
		return err
	}
	// oldname is the link target, not a path in this filesystem,
	// so we don't resolve it
	return os.Symlink(oldname, newpath)
}

// ReadLink implements fs.ReadLinkFS
func (f *FS) ReadLink(ctx context.Context, name string) (string, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return "", err
	}
	return os.Readlink(path)
}

// Lstat implements fs.LstatFS
func (f *FS) Lstat(ctx context.Context, name string) (fs.FileInfo, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

// Close removes the temporary directory if this filesystem was created with
// New(""). If the filesystem was created with a specific root directory,
// Close does nothing and returns nil.
//
// Close implements io.Closer.
func (f *FS) Close() error {
	if f.cleanupFn != nil {
		return f.cleanupFn()
	}
	return nil
}

// Compile-time interface checks
var (
	_ fs.FS         = (*FS)(nil)
	_ fs.CreateFS   = (*FS)(nil)
	_ fs.AppendFS   = (*FS)(nil)
	_ fs.RemoveFS   = (*FS)(nil)
	_ fs.MkdirFS    = (*FS)(nil)
	_ fs.RenameFS   = (*FS)(nil)
	_ fs.TruncateFS = (*FS)(nil)
	_ fs.ChtimesFS  = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.SymlinkFS  = (*FS)(nil)
	_ fs.ReadLinkFS = (*FS)(nil)
	_ io.Closer     = (*FS)(nil)
)
