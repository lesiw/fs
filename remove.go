package fs

import (
	"context"
	"errors"
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
//
// Requires: [RemoveFS]
func Remove(ctx context.Context, fsys FS, name string) error {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return err
	}
	if rfs, ok := fsys.(RemoveFS); ok {
		if err := rfs.Remove(ctx, name); !errors.Is(err, ErrUnsupported) {
			return newPathError("remove", name, err)
		}
	}
	return &PathError{Op: "remove", Path: name, Err: ErrUnsupported}
}
