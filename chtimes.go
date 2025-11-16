package fs

import (
	"context"
	"time"
)

// A ChtimesFS is a file system with the Chtimes method.
type ChtimesFS interface {
	FS

	// Chtimes changes the access and modification times of the named file.
	Chtimes(ctx context.Context, name string, atime, mtime time.Time) error
}

// Chtimes changes the access and modification times of the named file.
// Analogous to: [os.Chtimes], touch -t, 9P Twstat.
func Chtimes(
	ctx context.Context, fsys FS, name string, atime, mtime time.Time,
) error {
	cfs, ok := fsys.(ChtimesFS)
	if !ok {
		return &PathError{
			Op:   "chtimes",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	return cfs.Chtimes(ctx, name, atime, mtime)
}
