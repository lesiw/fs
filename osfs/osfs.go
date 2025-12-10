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
	"path"
	"path/filepath"
	"strings"
	"time"

	"lesiw.io/fs"
	fspath "lesiw.io/fs/path"
)

// osFS implements lesiw.io/fs.FS using the OS filesystem.
// It supports all optional interfaces defined in lesiw.io/fs.
//
// All paths are resolved relative to the current working directory,
// or relative to a directory specified via fs.WithWorkDir in the context.
//
// osFS implements io.Closer. If created with TempFS(), Close() removes
// the temporary directory.
//
// To create an osFS, use FS() or TempFS().
type osFS struct {
	cwd       string         // virtual CWD (empty for FS(), set for TempFS())
	cleanupFn func() error
}

// FS returns a filesystem that operates on the OS filesystem.
// All paths are resolved relative to the process's current working directory,
// or relative to a directory specified via fs.WithWorkDir in the context.
func FS() fs.FS {
	return &osFS{}
}

// TempFS creates a temporary directory and returns a filesystem with its
// virtual working directory set to the temp directory.
//
// Call fs.Close() on the returned filesystem to remove the temporary
// directory.
//
// TempFS never returns an error. If OS temp directory creation fails,
// it falls back to a local randomized path that will be created on first use.
func TempFS() fs.FS {
	fsys := &osFS{}

	// Try to use OS temp directory
	tmpdir, err := os.MkdirTemp("", "osfs-")
	if err == nil {
		fsys.cwd = tmpdir
		fsys.cleanupFn = func() error { return os.RemoveAll(tmpdir) }
		return fsys
	}

	// Fallback: use local randomized path
	tmpdir = fmt.Sprintf("osfs-tmp-%d", time.Now().UnixNano())
	fsys.cwd = tmpdir
	fsys.cleanupFn = func() error { return os.RemoveAll(tmpdir) }
	return fsys
}

// resolvePath converts a relative fs path to an absolute OS path.
// If ctx contains a working directory via fs.WorkDir(), paths are resolved
// relative to that working directory.
func (f *osFS) resolvePath(ctx context.Context, name string) (string, error) {
	name = filepath.Clean(name)
	if filepath.IsAbs(name) {
		return name, nil
	}

	// Start with virtual CWD if set (for TempFS), otherwise use os.Getwd()
	base := f.cwd
	if base == "" {
		var err error
		base, err = os.Getwd()
		if err != nil {
			// If Getwd fails, use empty base
			base = ""
		}
	}

	// Apply WorkDir from context if present
	if workDir := fs.WorkDir(ctx); workDir != "" {
		workDir = filepath.FromSlash(workDir)
		// If WorkDir is absolute, use it directly (ignore base CWD)
		if filepath.IsAbs(workDir) {
			base = workDir
		} else if base != "" {
			// WorkDir is relative, join with CWD
			base = filepath.Join(base, workDir)
		} else {
			// No CWD available, use WorkDir as-is
			base = workDir
		}
	}

	if base == "" {
		// No CWD and no WorkDir - just return the name as-is
		return filepath.FromSlash(name), nil
	}

	return filepath.Join(base, filepath.FromSlash(name)), nil
}

var _ fs.FS = (*osFS)(nil)

func (f *osFS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

var _ fs.CreateFS = (*osFS)(nil)

func (f *osFS) Create(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	perm := fs.FileMode(ctx)
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
}

var _ fs.AppendFS = (*osFS)(nil)

func (f *osFS) Append(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	perm := fs.FileMode(ctx)
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, perm)
}

var _ fs.StatFS = (*osFS)(nil)

func (f *osFS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Stat(path)
}

var _ fs.ReadDirFS = (*osFS)(nil)

func (f *osFS) ReadDir(
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

var _ fs.RemoveFS = (*osFS)(nil)

func (f *osFS) Remove(ctx context.Context, name string) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

var _ fs.MkdirFS = (*osFS)(nil)

func (f *osFS) Mkdir(ctx context.Context, name string) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	perm := fs.DirMode(ctx)
	return os.Mkdir(path, perm)
}

var _ fs.RenameFS = (*osFS)(nil)

func (f *osFS) Rename(ctx context.Context, oldname, newname string) error {
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

var _ fs.TruncateFS = (*osFS)(nil)

func (f *osFS) Truncate(ctx context.Context, name string, size int64) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Truncate(path, size)
}

var _ fs.ChtimesFS = (*osFS)(nil)

func (f *osFS) Chtimes(
	ctx context.Context, name string, atime, mtime time.Time,
) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Chtimes(path, atime, mtime)
}

var _ fs.SymlinkFS = (*osFS)(nil)

func (f *osFS) Symlink(ctx context.Context, oldname, newname string) error {
	newpath, err := f.resolvePath(ctx, newname)
	if err != nil {
		return err
	}
	// oldname is the link target, not a path in this filesystem,
	// so we don't resolve it
	return os.Symlink(oldname, newpath)
}

var _ fs.ReadLinkFS = (*osFS)(nil)

func (f *osFS) ReadLink(ctx context.Context, name string) (string, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return "", err
	}
	return os.Readlink(path)
}

func (f *osFS) Lstat(ctx context.Context, name string) (fs.FileInfo, error) {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

var _ fs.LocalizeFS = (*osFS)(nil)

func (f *osFS) Localize(ctx context.Context, path string) (string, error) {
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

	// If path is absolute, it's already in OS format - return as-is
	if filepath.IsAbs(p) {
		return p, nil
	}

	// Check if this is a directory path
	if fspath.IsDir(p) {
		dir := fspath.Dir(p)
		dir = path.Clean(dir)
		base, err := filepath.Localize(dir)
		if err != nil {
			return "", err
		}
		return base + string(filepath.Separator), nil
	}
	p = path.Clean(p)
	return filepath.Localize(p)
}

var _ fs.AbsFS = (*osFS)(nil)

func (f *osFS) Abs(ctx context.Context, name string) (string, error) {
	// If already absolute, return as-is
	if filepath.IsAbs(name) {
		return filepath.Clean(name), nil
	}

	// Resolve relative to CWD + WorkDir
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return "", err
	}

	return filepath.Clean(path), nil
}

var _ fs.RelFS = (*osFS)(nil)

func (f *osFS) Rel(
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

var _ fs.TempFS = (*osFS)(nil)

func (f *osFS) Temp(ctx context.Context, name string) (string, error) {
	file, err := os.CreateTemp("", name+"-")
	if err != nil {
		return "", err
	}
	defer file.Close()
	return file.Name(), nil
}

var _ fs.TempDirFS = (*osFS)(nil)

func (f *osFS) TempDir(ctx context.Context, name string) (string, error) {
	dir, err := os.MkdirTemp("", name+"-")
	if err != nil {
		return "", err
	}
	return dir, nil
}

var _ io.Closer = (*osFS)(nil)

// Close removes the temporary directory if this filesystem was created with
// TempFS(). If the filesystem was created with FS(), Close does nothing and
// returns nil.
func (f *osFS) Close() error {
	if f.cleanupFn != nil {
		return f.cleanupFn()
	}
	return nil
}
