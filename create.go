package fs

import (
	"context"
	"errors"
	"io"
	"path"
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
// When the name ends with a trailing slash (/), empties the directory (or
// creates it if it doesn't exist) and returns a tar stream writer for
// extracting files into it. Otherwise, creates a regular file.
//
// For files: If the file already exists, it is truncated. If the file does
// not exist, it is created with mode 0644 (or the mode specified via
// [WithFileMode]).
//
// For directories: Empties the directory via [Truncate], then returns an
// append writer. This is equivalent to Truncate(name, 0) followed by
// Append(name).
//
// If the parent directory does not exist and the filesystem implements
// [MkdirFS], Create automatically creates the parent directories with
// mode 0755 (or the mode specified via [WithDirMode]).
//
// The returned [io.WriteCloser] must be closed when done.
func Create(
	ctx context.Context, fsys FS, name string,
) (io.WriteCloser, error) {
	// Check if this is a directory path (trailing slash)
	if len(name) > 0 && name[len(name)-1] == '/' {
		dirName := name[:len(name)-1]

		// Ensure directory exists if MkdirFS is supported
		// (otherwise, directories are virtual and created by tar extraction)
		if _, ok := fsys.(MkdirFS); ok {
			if err := MkdirAll(ctx, fsys, dirName); err != nil {
				return nil, err
			}

			// Truncate to empty the directory
			if err := Truncate(ctx, fsys, name, 0); err != nil {
				return nil, err
			}
		}

		// Return append writer (tar extraction will create files/dirs)
		return Append(ctx, fsys, name)
	}

	// Check if filesystem supports Create
	cfs, ok := fsys.(CreateFS)
	if !ok {
		return nil, &PathError{
			Op:   "create",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	f, err := cfs.Create(ctx, name)
	if err == nil {
		return f, nil
	}

	// If the error is ErrNotExist, try to create parent directories
	if !errors.Is(err, ErrNotExist) {
		return nil, err
	}

	// Check if filesystem supports mkdir
	if _, ok := fsys.(MkdirFS); !ok {
		return nil, err // Return original error if mkdir not supported
	}

	// Create parent directory
	dir := path.Dir(name)
	if dir == "." || dir == name {
		return nil, err // No parent to create
	}

	if mkdirErr := MkdirAll(ctx, fsys, dir); mkdirErr != nil {
		return nil, err // Return original error, not mkdir error
	}

	// Try again after creating parent
	return cfs.Create(ctx, name)
}
