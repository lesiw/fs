package fs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"lesiw.io/fs/path"
)

// A TempFS is a file system with the Temp method.
//
// If not implemented, Temp falls back to TempDirFS.TempDir (creating a
// file inside a temp directory), then to CreateFS (creating a file with a
// random name).
type TempFS interface {
	FS

	// Temp creates a temporary file and returns its path.
	// The name parameter is used to generate the filename
	// (implementation-specific). The file will be created in an
	// OS-appropriate temporary location.
	//
	// Returns the path to the created file. The caller will use Create
	// or Append to open it for writing.
	//
	// If the filesystem cannot determine an appropriate temp location,
	// it should return ErrUnsupported to trigger the fallback behavior.
	Temp(ctx context.Context, name string) (string, error)
}

// A TempDirFS is a file system with the TempDir method.
//
// If not implemented, TempDir falls back to creating a directory with a
// random name using MkdirFS.
type TempDirFS interface {
	FS

	// TempDir creates a temporary directory and returns its path.
	// The name parameter is used to generate the directory name
	// (implementation-specific). The directory will be created in an
	// OS-appropriate temporary location.
	//
	// Returns the path to the created directory. The caller will use
	// Append with trailing slash to open it for tar writing.
	//
	// If the filesystem cannot determine an appropriate temp location,
	// it should return ErrUnsupported to trigger the fallback behavior.
	TempDir(ctx context.Context, name string) (string, error)
}

// Temp creates a temporary file or directory.
// Analogous to: [os.CreateTemp], [os.MkdirTemp], mktemp.
//
// The returned [WritePathCloser] must be closed when done. Path() returns
// the full path to the created resource. The caller is responsible for
// removing the temporary resource when done (typically with [RemoveAll]).
//
// # Files
//
// Without a trailing separator, creates a temporary file.
// The name parameter serves as a prefix or pattern (implementation-specific).
// The file name will typically have the pattern: name-randomhex
//
// Requires: [TempFS] || [TempDirFS] || [CreateFS]
//
// # Directories
//
// With a trailing separator, creates a temporary directory and returns a tar
// stream writer for extracting files into it. The directory name will
// typically have the pattern: name-randomhex
//
// Requires: [TempDirFS] || [MkdirFS]
func Temp(ctx context.Context, fsys FS, name string) (WritePathCloser, error) {
	// Check if this is a directory path (trailing separator)
	if path.IsDir(name) {
		// Remove trailing separator
		dirName := path.Dir(name)
		return tempDir(ctx, fsys, dirName)
	}

	return tempFile(ctx, fsys, name)
}

// tempFile creates a temporary file, trying TempFS, then TempDirFS,
// then CreateFS fallback.
func tempFile(
	ctx context.Context, fsys FS, name string,
) (WritePathCloser, error) {
	// Try TempFS first
	if tfs, ok := fsys.(TempFS); ok {
		tempPath, err := tfs.Temp(ctx, name)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return nil, err
		}
		if err == nil {
			// File created, now open it for writing
			return Create(ctx, fsys, tempPath)
		}
		// Fall through to TempDirFS fallback if ErrUnsupported
	}

	// Try TempDirFS as fallback (create dir, then file inside)
	if tfs, ok := fsys.(TempDirFS); ok {
		dirPath, err := tfs.TempDir(ctx, name)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return nil, err
		}
		if err == nil {
			// Create a file inside the temp directory
			return tempFileInDir(ctx, fsys, dirPath, name)
		}
		// Fall through to CreateFS fallback if ErrUnsupported
	}

	// Final fallback: CreateFS with random name
	return tempFileFallback(ctx, fsys, name)
}

// tempDir creates a temporary directory, trying TempDirFS then MkdirFS.
func tempDir(
	ctx context.Context, fsys FS, name string,
) (WritePathCloser, error) {
	// Try TempDirFS first
	if tfs, ok := fsys.(TempDirFS); ok {
		dirPath, err := tfs.TempDir(ctx, name)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return nil, err
		}
		if err == nil {
			// Directory created, now open it for tar writing
			return Append(ctx, fsys, path.Join(dirPath, ""))
		}
		// Fall through to MkdirFS fallback if ErrUnsupported
	}

	// Final fallback: MkdirFS with random name
	return tempDirFallback(ctx, fsys, name)
}

// tempFileInDir creates a file inside a temporary directory.
// Returns the file writer directly - caller is responsible for cleanup.
func tempFileInDir(
	ctx context.Context, fsys FS, dirPath, name string,
) (WritePathCloser, error) {
	// Check if CreateFS is supported
	if _, ok := fsys.(CreateFS); !ok {
		return nil, &PathError{
			Op:   "temp",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	// Create a file inside with a unique name
	filename, err := generateTempName(name)
	if err != nil {
		return nil, &PathError{Op: "temp", Path: name, Err: err}
	}

	filePath := path.Join(dirPath, filename)
	return Create(ctx, fsys, filePath)
}

// tempFileFallback creates a temporary file using Create.
func tempFileFallback(
	ctx context.Context, fsys FS, name string,
) (WritePathCloser, error) {
	// Check if CreateFS is supported
	if _, ok := fsys.(CreateFS); !ok {
		return nil, &PathError{
			Op:   "temp",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	// Generate filename with random suffix
	filename, err := generateTempName(name)
	if err != nil {
		return nil, &PathError{Op: "temp", Path: name, Err: err}
	}

	// Create the file
	return Create(ctx, fsys, filename)
}

// tempDirFallback creates a temporary directory using Mkdir.
func tempDirFallback(
	ctx context.Context, fsys FS, name string,
) (WritePathCloser, error) {
	// Check if MkdirFS is supported
	if _, ok := fsys.(MkdirFS); !ok {
		return nil, &PathError{
			Op:   "temp",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	// Generate directory name with random suffix
	dirname, err := generateTempName(name)
	if err != nil {
		return nil, &PathError{Op: "temp", Path: name, Err: err}
	}

	// Create directory with mode 0700
	dirCtx := WithDirMode(ctx, 0700)
	if err := Mkdir(dirCtx, fsys, dirname); err != nil {
		return nil, err
	}

	// Return tar writer for the directory
	dirPath := path.Join(dirname, "")
	w, err := Append(ctx, fsys, dirPath)
	if err != nil {
		// Try to clean up the directory we just created
		_ = Remove(ctx, fsys, dirname)
		return nil, err
	}

	// Override Path() to return dirname without trailing slash
	return &pathOverride{
		WritePathCloser: w,
		pathOverride:    dirname,
	}, nil
}

// generateTempName creates a name with random suffix.
func generateTempName(name string) (string, error) {
	// Generate random suffix
	var randBytes [16]byte
	if _, err := rand.Read(randBytes[:]); err != nil {
		return "", err
	}
	randSuffix := hex.EncodeToString(randBytes[:])

	// Create name with pattern: name-randomhex
	// If name is empty, use "tmp" as default
	if name != "" {
		return name + "-" + randSuffix, nil
	}
	return "tmp-" + randSuffix, nil
}

// pathOverride wraps a WritePathCloser and overrides the Path() method.
type pathOverride struct {
	WritePathCloser
	pathOverride string
}

func (p *pathOverride) Path() string {
	return p.pathOverride
}
