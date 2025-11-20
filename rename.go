package fs

import (
	"context"
	"errors"
	"io"
)

// A RenameFS is a file system with the Rename method.
type RenameFS interface {
	FS

	// Rename renames (moves) oldname to newname.
	// If newname already exists and is not a directory, Rename replaces it.
	Rename(ctx context.Context, oldname, newname string) error
}

// Rename renames (moves) oldname to newname.
// Analogous to: [os.Rename], mv, 9P2000.u Trename.
// If newname already exists and is not a directory, Rename replaces it.
//
// If the filesystem does not implement [RenameFS], Rename falls back to
// copying oldname to newname and then removing oldname. This requires
// [CreateFS]
// and [RemoveFS] support.
func Rename(ctx context.Context, fsys FS, oldname, newname string) error {
	if rfs, ok := fsys.(RenameFS); ok {
		err := rfs.Rename(ctx, oldname, newname)
		if err == nil || !errors.Is(err, ErrUnsupported) {
			return err
		}
		// Fall through to fallback if ErrUnsupported
	}

	// Fallback: copy file and delete original
	cfs, createOK := fsys.(CreateFS)
	rfs, removeOK := fsys.(RemoveFS)
	if !createOK || !removeOK {
		return &PathError{
			Op:   "rename",
			Path: oldname,
			Err:  ErrUnsupported,
		}
	}

	// Open source file
	src, err := fsys.Open(ctx, oldname)
	if err != nil {
		return &PathError{
			Op:   "rename",
			Path: oldname,
			Err:  err,
		}
	}
	defer src.Close()

	// Create destination file
	dst, err := cfs.Create(ctx, newname)
	if err != nil {
		return &PathError{
			Op:   "rename",
			Path: newname,
			Err:  err,
		}
	}

	// Copy data
	_, err = io.Copy(dst, src)
	closeErr := dst.Close()
	if err != nil {
		return &PathError{
			Op:   "rename",
			Path: newname,
			Err:  err,
		}
	}
	if closeErr != nil {
		return &PathError{
			Op:   "rename",
			Path: newname,
			Err:  closeErr,
		}
	}

	// Remove original file
	if err := rfs.Remove(ctx, oldname); err != nil {
		return &PathError{
			Op:   "rename",
			Path: oldname,
			Err:  err,
		}
	}

	return nil
}
