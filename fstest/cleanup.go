package fstest

import (
	"context"
	"testing"

	"lesiw.io/fs"
)

// cleanup registers cleanup for a path using t.Cleanup.
func cleanup(ctx context.Context, t *testing.T, fsys fs.FS, path string) {
	t.Helper()
	t.Cleanup(func() {
		if err := fs.RemoveAll(ctx, fsys, path); err != nil {
			t.Errorf("cleanup: RemoveAll(%q): %v", path, err)
		}
	})
}
