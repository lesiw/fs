package fs

import (
	"context"
	"errors"
	"io"

	"lesiw.io/fs/path"
)

// A CreateFS is a file system with the Create method.
type CreateFS interface {
	FS

	// Create creates a new file or truncates an existing file for writing.
	// If the file already exists, it is truncated. If the file does not exist,
	// it is created with mode 0644 (or the mode specified via WithFileMode).
	//
	// The returned writer must be closed when done. The writer may also
	// implement io.Seeker, io.WriterAt, or other interfaces depending
	// on the implementation.
	Create(ctx context.Context, name string) (io.WriteCloser, error)
}

// Create creates or truncates the named file for writing.
// Analogous to: [os.Create], touch, echo >, tar, 9P Tcreate, S3 PutObject.
//
// If the parent directory does not exist and the filesystem implements
// [MkdirFS], Create automatically creates the parent directories with
// mode 0755 (or the mode specified via [WithDirMode]).
//
// The returned [WritePathCloser] must be closed when done. The Path() method
// returns the native filesystem path, or the input path if localization is
// not supported.
//
// # Files
//
// If the file already exists, it is truncated. If the file does not exist,
// it is created with mode 0644 (or the mode specified via [WithFileMode]).
//
// Requires: [CreateFS]
//
// # Directories
//
// A trailing slash empties the directory (or creates it if it doesn't exist)
// and returns a tar stream writer for extracting files into it. This is
// equivalent to Truncate(name, 0) followed by Append(name).
//
// Requires: See [Truncate] and [Append] requirements
func Create(
	ctx context.Context, fsys FS, name string,
) (WritePathCloser, error) {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return nil, err
	}

	if path.IsDir(name) {
		w, err := createDirAsTar(ctx, fsys, name)
		if err != nil {
			return nil, err
		}
		return writePathCloser(w, name), nil
	}

	cfs, ok := fsys.(CreateFS)
	if !ok {
		return nil, &PathError{
			Op:   "create",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

retry:
	f, err := cfs.Create(ctx, name)
	if err == nil {
		return writePathCloser(f, name), nil
	}
	if errors.Is(err, ErrNotExist) {
		dir := path.Dir(name)
		if dir == "." || dir == name {
			return nil, err
		}
		if merr := MkdirAll(ctx, fsys, dir); merr != nil {
			return nil, errors.Join(err, merr)
		}
		goto retry
	}
	return writePathCloser(f, name), nil
}

func createDirAsTar(
	ctx context.Context, fsys FS, dir string,
) (io.WriteCloser, error) {
	dir = path.Dir(dir)
	if _, ok := fsys.(MkdirFS); ok {
		if err := MkdirAll(ctx, fsys, dir); err != nil {
			return nil, err
		}
		if err := Truncate(ctx, fsys, path.Join(dir, ""), 0); err != nil {
			return nil, err
		}
	}
	return Append(ctx, fsys, path.Join(dir, ""))
}
