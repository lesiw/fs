package fs

import (
	"archive/tar"
	"context"
	"errors"
	"io"

	"lesiw.io/fs/path"
)

// An AppendFS is a file system with the Append method.
type AppendFS interface {
	FS

	// Append opens a file for appending. Writes are added to the
	// end of the file.
	// If the file does not exist, it is created with mode 0644 (or the mode
	// specified via WithFileMode).
	//
	// The returned writer must be closed when done.
	Append(ctx context.Context, name string) (io.WriteCloser, error)
}

// An AppendDirFS is a file system that can write tar streams to directories.
//
// AppendDirFS is an optional interface that enables efficient bulk writes via
// tar archives, particularly useful for transferring many small files to
// remote filesystems. When not implemented, directory operations fall back to
// extracting tar archives file-by-file using [CreateFS].
type AppendDirFS interface {
	FS

	// AppendDir creates a tar stream for writing to the specified directory.
	// Data written to the returned writer is extracted as a tar archive into
	// the directory. The directory will be created if it doesn't exist.
	// Existing files with the same names will be overwritten, but other files
	// in the directory are preserved.
	//
	// The returned writer must be closed to complete the operation.
	AppendDir(ctx context.Context, dir string) (io.WriteCloser, error)
}

// Append opens a file for appending or adds files to a directory. Analogous
// to: [os.OpenFile] with O_APPEND, echo >>, tar (append mode), 9P Topen with
// OAPPEND.
//
// If the parent directory does not exist and the filesystem implements
// [MkdirFS], Append automatically creates the parent directories with mode
// 0755 (or the mode specified via [WithDirMode]).
//
// The returned [WritePathCloser] must be closed when done. The Path() method
// returns the native filesystem path, or the input path if localization is not
// supported.
//
// # Files
//
// Writes are added to the end of the file. If the file does not exist, it is
// created with mode 0644 (or the mode specified via [WithFileMode]).
//
// Requires: [AppendFS] || ([FS] && [CreateFS])
//
// # Directories
//
// A trailing slash returns a tar stream writer that extracts files into the
// directory. The directory is created if it doesn't exist. Existing files with
// the same names are overwritten, but other files in the directory are
// preserved.
//
// Requires: [AppendDirFS] || [CreateFS]
func Append(
	ctx context.Context, fsys FS, name string,
) (WritePathCloser, error) {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return nil, err
	}

	if path.IsDir(name) {
		w, err := appendDirAsTar(ctx, fsys, name)
		if err != nil {
			return nil, err
		}
		return writePathCloser(w, name), nil
	}

	afs, ok := fsys.(AppendFS)
	if !ok {
		if w, err := createAppend(ctx, fsys, name); err != nil {
			return nil, err
		} else {
			return writePathCloser(w, name), nil
		}
	}

retry:
	f, err := afs.Append(ctx, name)
	if err != nil {
		if !errors.Is(err, ErrNotExist) {
			return nil, err
		}
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

// createAppend implements append using CreateFS.
func createAppend(
	ctx context.Context, fsys FS, name string,
) (io.WriteCloser, error) {
	// Open existing file for reading, if it exists.
	r, err := Open(ctx, fsys, name)
	if err != nil && !errors.Is(err, ErrNotExist) {
		return nil, err
	}

	w, err := Create(ctx, fsys, name)
	if err != nil {
		if r != nil {
			_ = r.Close()
		}
		return nil, err
	}

	return newAppendWriter(r, w), nil
}

func appendDirAsTar(
	ctx context.Context, fsys FS, dir string,
) (io.WriteCloser, error) {
	dir = path.Dir(dir)
	if tfs, ok := fsys.(AppendDirFS); ok {
		w, err := tfs.AppendDir(ctx, dir)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return nil, err
		}
		if err == nil {
			return w, nil
		}
	}

	// Fallback: Extract one file at a time.
	pr, pw := io.Pipe()
	go func() {
		err := extractTarToFS(ctx, fsys, dir, pr)
		if err == nil {
			// Drain trailing data (e.g. tar block-alignment padding)
			// so the writer side doesn't get a broken pipe error.
			_, err = io.Copy(io.Discard, pr)
		}
		pr.CloseWithError(err)
	}()
	return pw, nil
}

// extractTarToFS reads a tar archive and extracts it to the filesystem.
func extractTarToFS(
	ctx context.Context, fsys FS, dir string, r io.Reader,
) error {
	tr := tar.NewReader(r)
	_, supportsMkdir := fsys.(MkdirFS)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Construct full path
		fullPath := path.Join(dir, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			// Only create directory if MkdirFS is supported
			// (otherwise directories are virtual)
			if supportsMkdir {
				dirCtx := WithDirMode(ctx, Mode(hdr.Mode))
				err = MkdirAll(dirCtx, fsys, fullPath)
				if err != nil {
					return err
				}
			}
		case tar.TypeReg:
			// Create parent directories only if MkdirFS is supported
			// (otherwise directories are virtual and created implicitly)
			if supportsMkdir {
				parent := path.Dir(fullPath)
				if err := MkdirAll(ctx, fsys, parent); err != nil {
					return err
				}
			}

			// Create file with mode from tar header
			fileCtx := WithFileMode(ctx, Mode(hdr.Mode))
			f, err := Create(fileCtx, fsys, fullPath)
			if err != nil {
				return err
			}

			// Copy contents
			_, copyErr := io.Copy(f, tr)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
}
