package fs

import (
	"context"
	"errors"
	"io"

	"lesiw.io/fs/path"
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
// A trailing slash indicates a directory. Removes all contents, leaving an
// empty directory.
//
// Requires: [TruncateDirFS] || ([RemoveAllFS] && [MkdirFS])
func Truncate(ctx context.Context, fsys FS, name string, size int64) error {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return err
	}

	if path.IsDir(name) {
		return truncateDirAsTar(ctx, fsys, name, size)
	}

	tfs, ok := fsys.(TruncateFS)
	if !ok {
		return recreateTruncate(ctx, fsys, name, size)
	}

	err = tfs.Truncate(ctx, name, size)
	if err == nil || !errors.Is(err, ErrUnsupported) {
		return err
	}
	return recreateTruncate(ctx, fsys, name, size)
}

func recreateTruncate(
	ctx context.Context, fsys FS, name string, size int64,
) error {
	// Special case: size 0 means create empty file
	if size == 0 {
		if err := Remove(ctx, fsys, name); err != nil {
			return &PathError{
				Op:   "truncate",
				Path: name,
				Err:  err,
			}
		}
		w, err := Create(ctx, fsys, name)
		if err != nil {
			return &PathError{
				Op:   "truncate",
				Path: name,
				Err:  err,
			}
		}
		return w.Close()
	}

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
	content = content[:n]
	if int64(n) < size {
		content = append(content, make([]byte, size-int64(n))...)
	}

	if err := Remove(ctx, fsys, name); err != nil {
		return &PathError{
			Op:   "truncate",
			Path: name,
			Err:  err,
		}
	}

	w, err := Create(ctx, fsys, name)
	if err != nil {
		return &PathError{
			Op:   "truncate",
			Path: name,
			Err:  err,
		}
	}
	if _, err := w.Write(content); err != nil {
		_ = w.Close()
		return &PathError{
			Op:   "truncate",
			Path: name,
			Err:  err,
		}
	}
	return w.Close()
}

func truncateDirAsTar(
	ctx context.Context, fsys FS, dir string, size int64,
) error {
	dir = path.Dir(dir)
	if size != 0 {
		return &PathError{
			Op:   "truncate",
			Path: dir,
			Err:  errors.New("directory truncate requires size 0"),
		}
	}

	tfs, ok := fsys.(TruncateDirFS)
	if !ok {
		return recreateTruncateDir(ctx, fsys, dir)
	}

	err := tfs.TruncateDir(ctx, dir)
	if err == nil || !errors.Is(err, ErrUnsupported) {
		return err
	}
	return recreateTruncateDir(ctx, fsys, dir)
}

func recreateTruncateDir(
	ctx context.Context, fsys FS, dir string,
) error {
	if sfs, ok := fsys.(StatFS); ok {
		info, err := sfs.Stat(ctx, dir)
		if err != nil {
			return &PathError{Op: "truncate", Path: dir, Err: err}
		}
		if !info.IsDir() {
			return &PathError{
				Op:   "truncate",
				Path: dir,
				Err:  errors.New("not a directory"),
			}
		}
	}

	if _, ok := fsys.(RemoveAllFS); ok {
		if err := RemoveAll(ctx, fsys, dir); err != nil {
			return &PathError{Op: "truncate", Path: dir, Err: err}
		}
	}

	if _, ok := fsys.(MkdirFS); ok {
		return MkdirAll(ctx, fsys, dir)
	}

	return &PathError{Op: "truncate", Path: dir, Err: ErrUnsupported}
}
