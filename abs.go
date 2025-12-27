package fs

import (
	"context"
	"errors"

	"lesiw.io/fs/path"
)

// An AbsFS is a file system with the Abs method.
type AbsFS interface {
	FS

	// Abs returns an absolute representation of path.
	// If the path is not absolute, it will be joined with the filesystem's
	// root and any working directory from ctx to produce an absolute path.
	//
	// The returned path format depends on the filesystem implementation:
	// local filesystems return OS paths, remote filesystems may return URLs.
	Abs(ctx context.Context, name string) (string, error)
}

// Abs returns an absolute representation of path within the filesystem.
//
// If the path is already absolute, Abs returns it cleaned.
// If the path is relative, Abs attempts to resolve it to an absolute path.
//
// The returned path format depends on the filesystem implementation.
// Local filesystems return OS-specific absolute paths (e.g., /home/user/file
// or C:\Users\file). Remote filesystems may return URLs (e.g.,
// s3://bucket/key or https://server/path).
//
// # Files
//
// Returns an absolute representation of the file path.
//
// Requires: [AbsFS] || (absolute [WorkDir] in ctx)
//
// # Directories
//
// Returns an absolute representation of the directory path.
//
// Requires: [AbsFS] || (absolute [WorkDir] in ctx)
//
// Similar capabilities: [path/filepath.Abs], realpath, pwd.
func Abs(ctx context.Context, fsys FS, name string) (string, error) {
	// Try native capability first
	if afs, ok := fsys.(AbsFS); ok {
		abs, err := afs.Abs(ctx, name)
		if !errors.Is(err, ErrUnsupported) {
			return abs, err
		}
	}

	// If path is already absolute, return it cleaned
	if path.IsAbs(name) {
		return path.Clean(name), nil
	}

	// Fallback: if WorkDir is absolute and path is relative, compute it
	if workDir := WorkDir(ctx); workDir != "" && path.IsAbs(workDir) {
		return path.Join(workDir, name), nil
	}

	return "", &PathError{Op: "abs", Path: name, Err: ErrUnsupported}
}
