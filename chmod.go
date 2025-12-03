package fs

import (
	"context"
	"errors"
)

// A ChmodFS is a file system with the Chmod method.
type ChmodFS interface {
	FS

	// Chmod changes the mode of the named file to mode.
	Chmod(ctx context.Context, name string, mode Mode) error
}

// Chmod changes the mode of the named file to mode.
// Analogous to: [os.Chmod], chmod, 9P Twstat.
//
// Requires: [ChmodFS]
func Chmod(
	ctx context.Context, fsys FS, name string, mode Mode,
) error {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return err
	}
	if cfs, ok := fsys.(ChmodFS); ok {
		if err := cfs.Chmod(ctx, name, mode); !errors.Is(err, ErrUnsupported) {
			return newPathError("chmod", name, err)
		}
	}
	return &PathError{Op: "chmod", Path: name, Err: ErrUnsupported}
}
