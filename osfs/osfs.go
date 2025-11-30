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
	"strings"
	"time"

	"lesiw.io/fs"
	fspath "lesiw.io/fs/path"
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

var _ fs.FS = (*FS)(nil)

func (f *FS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

var _ fs.CreateFS = (*FS)(nil)

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

var _ fs.AppendFS = (*FS)(nil)

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

var _ fs.StatFS = (*FS)(nil)

func (f *FS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Stat(path)
}

var _ fs.ReadDirFS = (*FS)(nil)

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

var _ fs.RemoveFS = (*FS)(nil)

func (f *FS) Remove(ctx context.Context, name string) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

var _ fs.MkdirFS = (*FS)(nil)

func (f *FS) Mkdir(ctx context.Context, name string) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	perm := fs.DirMode(ctx)
	return os.Mkdir(path, perm)
}

var _ fs.RenameFS = (*FS)(nil)

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

var _ fs.TruncateFS = (*FS)(nil)

func (f *FS) Truncate(ctx context.Context, name string, size int64) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Truncate(path, size)
}

var _ fs.ChtimesFS = (*FS)(nil)

func (f *FS) Chtimes(
	ctx context.Context, name string, atime, mtime time.Time,
) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Chtimes(path, atime, mtime)
}

var _ fs.SymlinkFS = (*FS)(nil)

func (f *FS) Symlink(ctx context.Context, oldname, newname string) error {
	newpath, err := f.resolvePath(ctx, newname)
	if err != nil {
		return err
	}
	// oldname is the link target, not a path in this filesystem,
	// so we don't resolve it
	return os.Symlink(oldname, newpath)
}

var _ fs.ReadLinkFS = (*FS)(nil)

func (f *FS) ReadLink(ctx context.Context, name string) (string, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return "", err
	}
	return os.Readlink(path)
}

func (f *FS) Lstat(ctx context.Context, name string) (fs.FileInfo, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

var _ fs.LocalizeFS = (*FS)(nil)

func (f *FS) Localize(ctx context.Context, path string) (string, error) {
	return localizePath(path)
}

// localizePath converts a Unix-style path to OS-specific format.
// Handles directory paths (indicated by trailing separator).
// This function is idempotent - calling it multiple times returns the same
// result, even though filepath.Localize itself is not idempotent on Windows.
func localizePath(p string) (string, error) {
	// If path is already localized (contains OS separator on Windows),
	// return as-is for idempotency. On Windows, backslashes indicate
	// the path was already localized.
	if filepath.Separator == '\\' && strings.ContainsRune(p, '\\') {
		return p, nil
	}

	// Check if this is a directory path
	if fspath.IsDir(p) {
		// Directory path - localize without trailing slash, then add it back
		dir := fspath.Dir(p)
		base, err := filepath.Localize(dir)
		if err != nil {
			return "", err
		}
		return base + string(filepath.Separator), nil
	}
	return filepath.Localize(p)
}

var _ fs.AbsFS = (*FS)(nil)

func (f *FS) Abs(ctx context.Context, name string) (string, error) {
	// If already absolute, return as-is
	if filepath.IsAbs(name) {
		return filepath.Clean(name), nil
	}

	// Resolve relative to root + WorkDir
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return "", err
	}

	return filepath.Clean(path), nil
}

var _ fs.RelFS = (*FS)(nil)

func (f *FS) Rel(
	ctx context.Context, basepath, targpath string,
) (string, error) {
	// Use filepath.Rel for OS-specific path handling
	rel, err := filepath.Rel(basepath, targpath)
	if err != nil {
		return "", err
	}
	// Convert to forward slashes for fs consistency
	return filepath.ToSlash(rel), nil
}

var _ io.Closer = (*FS)(nil)

// Close removes the temporary directory if this filesystem was created with
// New(""). If the filesystem was created with a specific root directory,
// Close does nothing and returns nil.
func (f *FS) Close() error {
	if f.cleanupFn != nil {
		return f.cleanupFn()
	}
	return nil
}
