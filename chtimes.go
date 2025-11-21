package fs

import (
	"context"
	"errors"
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
//
// Requires: [ChtimesFS]
func Chtimes(
	ctx context.Context, fsys FS, name string, atime, mtime time.Time,
) error {
	if cfs, ok := fsys.(ChtimesFS); ok {
		err := cfs.Chtimes(ctx, name, atime, mtime)
		if !errors.Is(err, ErrUnsupported) {
			return newPathError("chtimes", name, err)
		}
	}
	return &PathError{Op: "chtimes", Path: name, Err: ErrUnsupported}
}
