package fs

import (
	"archive/tar"
	"context"
	"errors"
	"io"
	"path"
	"strings"
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
// The returned [io.ReadCloser] must be closed when done.
//
// # Files
//
// Returns an [io.ReadCloser] for reading the file contents.
//
// Requires: [FS]
//
// # Directories
//
// A trailing slash (/) or a path identified as a directory via [StatFS]
// returns a tar archive stream of the directory contents.
//
// Requires: [DirFS] || ([FS] && ([ReadDirFS] || [WalkFS]))
func Open(ctx context.Context, fsys FS, name string) (io.ReadCloser, error) {
	// Check if name has trailing slash - indicates directory
	if len(name) > 0 && name[len(name)-1] == '/' {
		dirName := name[:len(name)-1]
		return openDirAsTar(ctx, fsys, dirName)
	}

	// Check if it's a directory via stat
	if sfs, ok := fsys.(StatFS); ok {
		info, err := sfs.Stat(ctx, name)
		if err == nil && info.IsDir() {
			return openDirAsTar(ctx, fsys, name)
		}
	}

	// Regular file open
	return fsys.Open(ctx, name)
}

// openDirAsTar opens a tar stream for reading from dir in fsys.
// If fsys implements DirFS, uses the native implementation.
// If the native implementation returns ErrUnsupported, falls back to walking
// the directory and creating tar manually.
//
// The returned reader must be closed when done.
func openDirAsTar(
	ctx context.Context, fsys FS, dir string,
) (io.ReadCloser, error) {
	if tfs, ok := fsys.(DirFS); ok {
		r, err := tfs.OpenDir(ctx, dir)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return nil, err
		}
		if err == nil {
			return r, nil
		}
		// Fall through to fallback if ErrUnsupported
	}
	return openTarFallback(ctx, fsys, dir)
}

// openTarFallback creates a tar archive by walking the filesystem.
//
// The returned ReadCloser must be closed when done reading. The spawned
// goroutine will terminate when the reader is closed or when the entire
// tar archive has been read to EOF.
func openTarFallback(
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
