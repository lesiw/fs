package fs

import (
	"context"
	"errors"
	"path"
)

// A RemoveFS is a file system with the Remove method.
type RemoveFS interface {
	FS

	// Remove removes the named file or empty directory.
	// It returns an error if the file does not exist or if a directory
	// is not empty.
	Remove(ctx context.Context, name string) error
}

// Remove removes the named file or empty directory.
// Analogous to: [os.Remove], rm, 9P Tremove, S3 DeleteObject.
// Returns an error if the file does not exist or if a directory is not
// empty.
func Remove(ctx context.Context, fsys FS, name string) error {
	rfs, ok := fsys.(RemoveFS)
	if !ok {
		return &PathError{
			Op:   "remove",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	return rfs.Remove(ctx, name)
}

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
// If fsys implements [RemoveAllFS], RemoveAll uses the native implementation.
// Otherwise, RemoveAll falls back to recursive removal using [RemoveFS],
// [StatFS], and [ReadDirFS].
func RemoveAll(ctx context.Context, fsys FS, name string) error {
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
	err := rfs.Remove(ctx, name)
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
