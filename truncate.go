package fs

import (
	"context"
	"errors"
)

// A TruncateFS is a file system with the Truncate method.
type TruncateFS interface {
	FS

	// Truncate changes the size of the named file.
	// If the file is larger than size, it is truncated.
	// If it is smaller, it is extended with zeros.
	Truncate(ctx context.Context, name string, size int64) error
}

// A TruncateDirFS is a file system that can efficiently empty directories.
//
// TruncateDirFS is an optional interface for efficient directory truncation.
// When not implemented, falls back to [RemoveAll] + [Mkdir] when available.
type TruncateDirFS interface {
	FS

	// TruncateDir removes all contents from the specified directory, leaving
	// it empty. Returns an error if the directory doesn't exist.
	TruncateDir(ctx context.Context, dir string) error
}

// Truncate changes the size of the named file or empties a directory.
// Analogous to: [os.Truncate], truncate, 9P Twstat.
//
// For files: If the file is larger than size, it is truncated. If it is
// smaller, it is extended with zeros.
//
// For directories (trailing slash): Removes all contents, leaving an empty
// directory.
//
// Like [os.Truncate], Truncate returns an error if the path doesn't exist.
// If [StatFS] is available, the existence check happens before attempting the
// operation. Otherwise, the error comes from the truncate operation itself.
//
// If the filesystem does not implement [TruncateFS] (or [TruncateDirFS] for
// directories) and size is 0, Truncate falls back to emptying via [Create]
// for files or [RemoveAll]+[Mkdir] for directories.
func Truncate(ctx context.Context, fsys FS, name string, size int64) error {
	// Check if this is a directory (trailing slash)
	if len(name) > 0 && name[len(name)-1] == '/' {
		dirName := name[:len(name)-1]
		return truncateDirAsTar(ctx, fsys, dirName, size)
	}

	// Try native Truncate first
	if tfs, ok := fsys.(TruncateFS); ok {
		return tfs.Truncate(ctx, name, size)
	}

	// Fallback: if size is 0, check existence first if possible
	if size == 0 {
		// If we can stat, verify the file exists
		if sfs, ok := fsys.(StatFS); ok {
			_, err := sfs.Stat(ctx, name)
			if err != nil {
				// File doesn't exist or other error - return it
				return &PathError{
					Op:   "truncate",
					Path: name,
					Err:  err,
				}
			}
		}

		// File exists (or we can't check) - truncate via Create
		f, err := Create(ctx, fsys, name)
		if err != nil {
			return err
		}
		return f.Close()
	}

	// No fallback for non-zero sizes
	return &PathError{
		Op:   "truncate",
		Path: name,
		Err:  ErrUnsupported,
	}
}

// truncateDirAsTar empties a directory.
// If fsys implements TruncateDirFS, uses the native TruncateDir
// implementation.
// Otherwise, falls back to RemoveAll + Mkdir when available.
// Returns an error if the directory doesn't exist.
func truncateDirAsTar(
	ctx context.Context, fsys FS, dir string, size int64,
) error {
	if size != 0 {
		return &PathError{
			Op:   "truncate",
			Path: dir,
			Err:  errors.New("directory truncate requires size 0"),
		}
	}
	// Try native TruncateDir first
	if tfs, ok := fsys.(TruncateDirFS); ok {
		err := tfs.TruncateDir(ctx, dir)
		if err == nil || !errors.Is(err, ErrUnsupported) {
			return err
		}
		// Fall through to fallback if ErrUnsupported
	}

	// Fallback: Check existence, remove contents, recreate
	// First check if directory exists
	if sfs, ok := fsys.(StatFS); ok {
		info, err := sfs.Stat(ctx, dir)
		if err != nil {
			// Directory doesn't exist or error - return error
			return &PathError{Op: "truncate", Path: dir, Err: err}
		}
		if !info.IsDir() {
			// Path exists but is not a directory
			return &PathError{
				Op:   "truncate",
				Path: dir,
				Err:  errors.New("not a directory"),
			}
		}
	}

	// Directory exists - remove and recreate
	if _, ok := fsys.(RemoveAllFS); ok {
		if err := RemoveAll(ctx, fsys, dir); err != nil {
			return &PathError{Op: "truncate", Path: dir, Err: err}
		}
	}

	// Recreate empty directory (use MkdirAll for idempotency)
	if _, ok := fsys.(MkdirFS); ok {
		return MkdirAll(ctx, fsys, dir)
	}

	// Can't recreate - return unsupported
	return &PathError{Op: "truncate", Path: dir, Err: ErrUnsupported}
}
