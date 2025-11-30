package fs

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"strings"

	"lesiw.io/fs/path"
)

// An FS is a file system with the Open method.
type FS interface {
	// Open opens the named file for reading.
	//
	// The returned reader must be closed when done. The reader may also
	// implement io.Seeker, io.ReaderAt, or other interfaces depending
	// on the implementation.
	Open(ctx context.Context, name string) (io.ReadCloser, error)
}

// A DirFS is a file system that can read directories as tar streams.
//
// DirFS is an optional interface that enables efficient bulk reads via tar
// archives, particularly useful for read-only filesystems or transferring many
// small files from remote filesystems. When not implemented, directory
// operations fall back to walking the filesystem and creating tar archives
// manually.
type DirFS interface {
	FS

	// OpenDir opens a tar stream for reading from the specified directory.
	// The directory is archived as a tar stream that can be read until EOF.
	//
	// The returned reader must be closed when done.
	OpenDir(ctx context.Context, dir string) (io.ReadCloser, error)
}

// Open opens the named file or directory for reading.
// Analogous to: [io/fs.Open], [os.Open], cat, tar, 9P Topen, S3 GetObject.
//
// All paths use forward slashes (/) regardless of the operating system,
// following [io/fs] conventions. Use the [path] package (not [path/filepath])
// for path manipulation. Implementations handle OS-specific conversion
// internally.
//
// The returned [ReadPathCloser] must be closed when done. The Path() method
// returns the native filesystem path, or the input path if localization is
// not supported.
//
// # Files
//
// Returns a [ReadPathCloser] for reading the file contents.
//
// Requires: [FS]
//
// # Directories
//
// A trailing slash returns a tar archive stream of the directory contents.
// A path identified as a directory via [StatFS] also returns a tar archive.
//
// Requires: [DirFS] || ([FS] && ([ReadDirFS] || [WalkFS]))
func Open(ctx context.Context, fsys FS, name string) (ReadPathCloser, error) {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return nil, err
	}

	if path.IsDir(name) {
		r, err := openDirAsTar(ctx, fsys, name)
		if err != nil {
			return nil, err
		}
		return readPathCloser(r, name), nil
	}

	if sfs, ok := fsys.(StatFS); ok {
		info, err := sfs.Stat(ctx, name)
		if err == nil && info.IsDir() {
			r, err := openDirAsTar(ctx, fsys, name)
			if err != nil {
				return nil, err
			}
			return readPathCloser(r, name), nil
		}
	}

	r, err := fsys.Open(ctx, name)
	if err != nil {
		return nil, err
	}
	return readPathCloser(r, name), nil
}

func openDirAsTar(
	ctx context.Context, fsys FS, dir string,
) (io.ReadCloser, error) {
	dir = path.Dir(dir)
	if tfs, ok := fsys.(DirFS); ok {
		r, err := tfs.OpenDir(ctx, dir)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return nil, err
		}
		if err == nil {
			return r, nil
		}
	}
	return walkDirAsTar(ctx, fsys, dir)
}

func walkDirAsTar(
	ctx context.Context, fsys FS, dir string,
) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		err := createTarFromFS(ctx, fsys, dir, pw)
		pw.CloseWithError(err)
	}()

	return pr, nil
}

// createTarFromFS walks the filesystem and creates a tar archive.
func createTarFromFS(
	ctx context.Context, fsys FS, dir string, w io.Writer,
) error {
	tw := tar.NewWriter(w)
	defer tw.Close()

	// Walk all entries and add to tar
	var walkPath func(string, int) error
	walkPath = func(currentPath string, currentDepth int) error {
		for entry, err := range ReadDir(ctx, fsys, currentPath) {
			if err != nil {
				return err
			}

			// Build full path
			entryPath := path.Join(currentPath, entry.Name())

			// Get relative path from dir
			relPath := strings.TrimPrefix(entryPath, dir)
			relPath = strings.TrimPrefix(relPath, "/")

			// Get file info
			info, infoErr := entry.Info()
			if infoErr != nil {
				return infoErr
			}

			// Create tar header
			hdr, hdrErr := tar.FileInfoHeader(info, "")
			if hdrErr != nil {
				return hdrErr
			}
			hdr.Name = relPath

			// Write header
			if writeErr := tw.WriteHeader(hdr); writeErr != nil {
				return writeErr
			}

			// Write file contents if not a directory
			if !entry.IsDir() {
				f, openErr := Open(ctx, fsys, entryPath)
				if openErr != nil {
					return openErr
				}
				_, copyErr := io.Copy(tw, f)
				closeErr := f.Close()
				if copyErr != nil {
					return copyErr
				}
				if closeErr != nil {
					return closeErr
				}
			} else {
				// Recurse into subdirectory
				recurseErr := walkPath(entryPath, currentDepth+1)
				if recurseErr != nil {
					return recurseErr
				}
			}
		}
		return nil
	}

	return walkPath(dir, 0)
}
