package fs

import (
	"context"
	"errors"

	"lesiw.io/fs/path"
)

// A RemoveAllFS is a file system with the RemoveAll method.
//
// If not implemented, RemoveAll falls back to recursive removal using
// RemoveFS, StatFS, and ReadDirFS.
type RemoveAllFS interface {
	FS

	// RemoveAll removes name and any children it contains.
	RemoveAll(ctx context.Context, name string) error
}

// RemoveAll removes name and any children it contains.
// Analogous to: [os.RemoveAll], rm -rf.
//
// Requires: [RemoveAllFS] ||
// ([RemoveFS] && [StatFS] && ([ReadDirFS] || [WalkFS]))
func RemoveAll(ctx context.Context, fsys FS, name string) error {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return err
	}
	// Check for efficient RemoveAll implementation first
	if rafs, ok := fsys.(RemoveAllFS); ok {
		err := rafs.RemoveAll(ctx, name)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return err
		}
		if err == nil {
			return nil
		}
		// Fall through to fallback if ErrUnsupported
	}

	// Check if fallback is possible - requires RemoveFS, StatFS, ReadDirFS
	rfs, hasRemove := fsys.(RemoveFS)
	_, hasStat := fsys.(StatFS)
	_, hasReadDir := fsys.(ReadDirFS)

	if !hasRemove || !hasStat || !hasReadDir {
		return &PathError{
			Op:   "remove",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	// Try to remove it directly first
	err = rfs.Remove(ctx, name)
	if err == nil || errors.Is(err, ErrNotExist) {
		return nil
	}

	// If removal failed, check if it's a directory with contents
	info, statErr := Stat(ctx, fsys, name)
	if statErr != nil {
		return statErr
	}

	if !info.IsDir() {
		return err
	}

	// It's a directory - read contents to remove children
	// Remove all children
	for entry, readErr := range ReadDir(ctx, fsys, name) {
		if readErr != nil {
			return readErr
		}
		childPath := path.Join(name, entry.Name())
		if removeErr := RemoveAll(ctx, fsys, childPath); removeErr != nil {
			return removeErr
		}
	}

	// Now remove the empty directory
	return rfs.Remove(ctx, name)
}
