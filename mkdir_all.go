package fs

import (
	"context"
	"errors"

	"lesiw.io/fs/path"
)

// A MkdirAllFS is a file system with the MkdirAll method.
//
// If not implemented, MkdirAll falls back to recursive creation using
// MkdirFS and StatFS.
type MkdirAllFS interface {
	FS

	// MkdirAll creates a directory named name, along with any necessary
	// parents. If name is already a directory, MkdirAll does nothing and
	// returns nil.
	MkdirAll(ctx context.Context, name string) error
}

// MkdirAll creates a directory named name, along with any necessary parents.
// Analogous to: [os.MkdirAll], mkdir -p.
//
// The directory mode is obtained from [DirMode](ctx). If not set in the
// context, the default mode 0755 is used:
//
//	ctx = fs.WithDirMode(ctx, 0700)
//	fs.MkdirAll(ctx, fsys, "a/b/c")  // All created with mode 0700
//
// If name is already a directory, MkdirAll does nothing and returns nil.
//
// Requires: [MkdirAllFS] || ([MkdirFS] && [StatFS])
func MkdirAll(ctx context.Context, fsys FS, name string) error {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return err
	}

	mafs, ok := fsys.(MkdirAllFS)
	if !ok {
		return mkdirAllFallback(ctx, fsys, name)
	}

	err = mafs.MkdirAll(ctx, name)
	if err != nil && !errors.Is(err, ErrUnsupported) {
		return err
	}
	if err == nil {
		return nil
	}

	// Fall through to fallback if ErrUnsupported
	return mkdirAllFallback(ctx, fsys, name)
}

// mkdirAllFallback implements MkdirAll using MkdirFS and StatFS.
func mkdirAllFallback(ctx context.Context, fsys FS, name string) error {
	// Check if fallback is possible - requires MkdirFS and StatFS
	mfs, hasMkdir := fsys.(MkdirFS)
	_, hasStat := fsys.(StatFS)

	if !hasMkdir || !hasStat {
		return &PathError{
			Op:   "mkdir",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	// Check if it already exists
	info, err := Stat(ctx, fsys, name)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return &PathError{
			Op:   "mkdir",
			Path: name,
			Err:  ErrNotDir,
		}
	}

	// Try to create the directory
	err = mfs.Mkdir(ctx, name)
	if err == nil || errors.Is(err, ErrExist) {
		return nil
	}

	// Mkdir failed - try to create parent directories
	// Most commonly this is because the parent doesn't exist,
	// but we recurse regardless of the error type
	parent := path.Dir(name)
	if parent == "." || parent == name {
		return err
	}

	// Recursively create parent
	if err := MkdirAll(ctx, fsys, parent); err != nil {
		return err
	}

	// Try again (ignore ErrExist in case created by parent)
	err = mfs.Mkdir(ctx, name)
	if err == nil || errors.Is(err, ErrExist) {
		return nil
	}
	return err
}
