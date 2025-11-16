package fs

import (
	"context"
)

// A ChmodFS is a file system with the Chmod method.
type ChmodFS interface {
	FS

	// Chmod changes the mode of the named file to mode.
	Chmod(ctx context.Context, name string, mode Mode) error
}

// Chmod changes the mode of the named file to mode.
// Analogous to: [os.Chmod], chmod, 9P Twstat.
func Chmod(
	ctx context.Context, fsys FS, name string, mode Mode,
) error {
	cfs, ok := fsys.(ChmodFS)
	if !ok {
		return &PathError{
			Op:   "chmod",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	return cfs.Chmod(ctx, name, mode)
}
