package fs

import (
	"context"
	"errors"
)

// A StatFS is a file system with the Stat method.
type StatFS interface {
	FS

	// Stat returns file metadata for the named file.
	Stat(ctx context.Context, name string) (FileInfo, error)
}

// Stat returns file metadata for the named file.
// Analogous to: [io/fs.Stat], [os.Stat], stat, ls -l, 9P Tstat,
// S3 HeadObject.
//
// Requires: [StatFS]
func Stat(ctx context.Context, fsys FS, name string) (FileInfo, error) {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return nil, err
	}
	if sfs, ok := fsys.(StatFS); ok {
		if info, err := sfs.Stat(ctx, name); !errors.Is(err, ErrUnsupported) {
			return info, newPathError("stat", name, err)
		}
	}
	return nil, &PathError{Op: "stat", Path: name, Err: ErrUnsupported}
}
