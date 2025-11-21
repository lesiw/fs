package fs

import (
	"context"
	"errors"
	"io"
)

// A ReadFileFS is a file system with the ReadFile method.
//
// If not implemented, ReadFile falls back to Open and [io.ReadAll].
type ReadFileFS interface {
	FS

	// ReadFile reads the named file and returns its contents.
	ReadFile(ctx context.Context, name string) ([]byte, error)
}

// ReadFile reads the named file and returns its contents.
// Analogous to: [io/fs.ReadFile], [os.ReadFile], cat.
//
// Requires: [ReadFileFS] || [FS]
func ReadFile(ctx context.Context, fsys FS, name string) ([]byte, error) {
	if rffs, ok := fsys.(ReadFileFS); ok {
		data, err := rffs.ReadFile(ctx, name)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return data, err
		}
		if err == nil {
			return data, nil
		}
		// Fall through to fallback if ErrUnsupported
	}
	// Fall back to opening and reading
	// (always possible - uses base FS interface)
	f, err := Open(ctx, fsys, name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
