package fs

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"slices"
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

// Append opens a file for appending or adds files to a directory.
// Analogous to: [os.OpenFile] with O_APPEND, echo >>, tar (append mode), 9P
// Topen with OAPPEND.
//
// For files: Writes are added to the end of the file. If the file does not
// exist, it is created with mode 0644 (or the mode specified via
// [WithFileMode]).
//
// For directories (trailing slash): Returns a tar stream writer that extracts
// files into the directory. The directory is created if it doesn't exist.
// Existing files with the same names are overwritten, but other files in the
// directory are preserved.
//
// If the parent directory does not exist and the filesystem implements
// [MkdirFS], Append automatically creates the parent directories with
// mode 0755 (or the mode specified via [WithDirMode]).
//
// If the filesystem does not implement [AppendFS] (or [DirFS] for
// directories), Append falls back to reading the existing file (if it exists)
// and creating a new file with the combined content on Close().
//
// The returned [io.WriteCloser] must be closed when done.
func Append(
	ctx context.Context, fsys FS, name string,
) (io.WriteCloser, error) {
	// Check if this is a directory path (trailing slash)
	if len(name) > 0 && name[len(name)-1] == '/' {
		dirName := name[:len(name)-1]
		return appendDirAsTar(ctx, fsys, dirName)
	}
	// Check if filesystem supports Append natively
	if afs, ok := fsys.(AppendFS); ok {
		f, err := afs.Append(ctx, name)
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

		mkdirErr := MkdirAll(ctx, fsys, dir)
		if mkdirErr != nil {
			return nil, err // Return original error, not mkdir error
		}

		// Try again after creating parent
		return afs.Append(ctx, name)
	}

	// Fallback: use ReadFile + Create pattern
	return appendFallback(ctx, fsys, name)
}

// appendFallback implements append using ReadFile + Create
func appendFallback(
	ctx context.Context, fsys FS, name string,
) (io.WriteCloser, error) {
	// Read existing content (if file exists)
	existing, err := ReadFile(ctx, fsys, name)
	if err != nil && !errors.Is(err, ErrNotExist) {
		return nil, err
	}

	return &appendWriter{
		ctx:      ctx,
		fsys:     fsys,
		name:     name,
		existing: existing,
		buf:      &bytes.Buffer{},
	}, nil
}

// appendWriter buffers writes and combines with existing content on Close
type appendWriter struct {
	ctx      context.Context
	fsys     FS
	name     string
	existing []byte
	buf      *bytes.Buffer
}

func (w *appendWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *appendWriter) Close() error {
	// Combine existing content + new writes
	combined := slices.Concat(w.existing, w.buf.Bytes())

	// Write combined content
	return WriteFile(w.ctx, w.fsys, w.name, combined)
}

// appendDirAsTar creates a tar stream for appending to dir in fsys.
// If fsys implements AppendDirFS, uses the native implementation.
// If the native implementation returns ErrUnsupported, falls back to
// extracting files individually using archive/tar.
//
// The returned writer must be closed to complete the extraction.
func appendDirAsTar(
	ctx context.Context, fsys FS, dir string,
) (io.WriteCloser, error) {
	if tfs, ok := fsys.(AppendDirFS); ok {
		w, err := tfs.AppendDir(ctx, dir)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return nil, err
		}
		if err == nil {
			return w, nil
		}
		// Fall through to fallback if ErrUnsupported
	}
	return appendTarFallback(ctx, fsys, dir)
}

// appendTarFallback extracts a tar archive to the filesystem.
//
// The returned WriteCloser must be closed to signal completion and allow
// the extraction goroutine to terminate. Errors from extraction are
// reported when Close() is called on the returned writer.
func appendTarFallback(
	ctx context.Context, fsys FS, dir string,
) (io.WriteCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		err := extractTarToFS(ctx, fsys, dir, pr)
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
