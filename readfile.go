package fs

import (
	"context"
	"io"
)

// ReadFile reads the named file and returns its contents.
// Analogous to: [io/fs.ReadFile], [os.ReadFile], cat.
//
// Requires: [FS]
func ReadFile(ctx context.Context, fsys FS, name string) ([]byte, error) {
	f, err := Open(ctx, fsys, name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}
