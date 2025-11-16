package fs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
)

// TempDir creates a temporary directory.
// Analogous to: [os.MkdirTemp], mktemp -d.
//
// The directory name will have the pattern prefix-randomhex.
// The caller is responsible for removing the directory when done.
//
// If fsys implements [TempFS], TempDir uses the native implementation.
// Otherwise, TempDir falls back to creating a directory with a random
// name in the current directory (requires [MkdirFS]).
func TempDir(ctx context.Context, fsys FS, prefix string) (string, error) {
	if tfs, ok := fsys.(TempFS); ok {
		dir, err := tfs.TempDir(ctx, prefix)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return "", err
		}
		if err == nil {
			return dir, nil
		}
		// Fall through to fallback if ErrUnsupported
	}

	// Check if fallback is possible - requires MkdirFS
	if _, ok := fsys.(MkdirFS); !ok {
		return "", &PathError{
			Op:   "tempdir",
			Path: prefix,
			Err:  ErrUnsupported,
		}
	}

	return tempDirFallback(ctx, fsys, prefix)
}

// tempDirFallback creates a temporary directory using mkdir.
func tempDirFallback(
	ctx context.Context, fsys FS, prefix string,
) (string, error) {
	// Generate random suffix
	var randBytes [16]byte
	if _, err := rand.Read(randBytes[:]); err != nil {
		return "", &PathError{Op: "tempdir", Path: prefix, Err: err}
	}
	randSuffix := hex.EncodeToString(randBytes[:])

	// Create directory name with pattern: prefix-randomhex
	// If prefix is empty, use "tmp" as default
	var dirname string
	if prefix != "" {
		dirname = prefix + "-" + randSuffix
	} else {
		dirname = "tmp-" + randSuffix
	}

	// Try to create in current directory with mode 0700
	dirCtx := WithDirMode(ctx, 0700)
	err := Mkdir(dirCtx, fsys, dirname)
	if err != nil {
		return "", &PathError{Op: "tempdir", Path: dirname, Err: err}
	}

	return dirname, nil
}

// A TempFS is a file system with the TempDir method.
//
// If not implemented, TempDir falls back to creating a directory with a
// random name using MkdirFS.
type TempFS interface {
	FS

	// TempDir creates a temporary directory with the given prefix.
	// The directory will be created in an OS-appropriate temporary location.
	// The full path to the created directory is returned.
	//
	// If the filesystem cannot determine an appropriate temp location,
	// it should return ErrUnsupported to trigger the fallback behavior.
	TempDir(ctx context.Context, prefix string) (string, error)
}
