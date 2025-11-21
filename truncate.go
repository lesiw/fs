package fs

import (
	"context"
	"errors"
	"io"
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
// Like [os.Truncate], Truncate returns an error if the path doesn't exist.
// If [StatFS] is available, the existence check happens before attempting the
// operation. Otherwise, the error comes from the truncate operation itself.
//
// # Files
//
// If the file is larger than size, it is truncated. If it is smaller, it is
// extended with zeros.
//
// Requires: [TruncateFS] || ([FS] && [RemoveFS] && [CreateFS])
//
// # Directories
//
// A trailing slash (/) indicates a directory. Removes all contents, leaving
// an empty directory.
//
// Requires: [TruncateDirFS] || ([RemoveAllFS] && [MkdirFS])
func Truncate(ctx context.Context, fsys FS, name string, size int64) error {
	// Check if this is a directory (trailing slash)
	if len(name) > 0 && name[len(name)-1] == '/' {
		dirName := name[:len(name)-1]
		return truncateDirAsTar(ctx, fsys, dirName, size)
	}

	// Try native Truncate first
	if tfs, ok := fsys.(TruncateFS); ok {
		err := tfs.Truncate(ctx, name, size)
		if err == nil || !errors.Is(err, ErrUnsupported) {
			return err
		}
		// Fall through to fallback if ErrUnsupported
	}

	// Fallback: read existing content, remove, recreate with truncated content
	// Read up to 'size' bytes from the existing file
	f, err := Open(ctx, fsys, name)
	if err != nil {
		return &PathError{
			Op:   "truncate",
			Path: name,
			Err:  err,
		}
	}
	content := make([]byte, size)
	n, readErr := io.ReadFull(f, content)
	closeErr := f.Close()
	if closeErr != nil {
		return &PathError{
			Op:   "truncate",
			Path: name,
			Err:  closeErr,
		}
	}
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		return &PathError{
			Op:   "truncate",
			Path: name,
			Err:  readErr,
		}
	}
	// If file was smaller than size, extend with zeros
	content = content[:n]
	if int64(n) < size {
		content = append(content, make([]byte, size-int64(n))...)
	}

	// Remove the file
	if err := Remove(ctx, fsys, name); err != nil {
		return &PathError{
			Op:   "truncate",
			Path: name,
			Err:  err,
		}
	}

	// Create new file with truncated content
	w, err := Create(ctx, fsys, name)
	if err != nil {
		return &PathError{
			Op:   "truncate",
			Path: name,
			Err:  err,
		}
	}
	if len(content) > 0 {
		if _, err := w.Write(content); err != nil {
			_ = w.Close()
			return &PathError{
				Op:   "truncate",
				Path: name,
				Err:  err,
			}
		}
	}
	return w.Close()
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
