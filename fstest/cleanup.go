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
		// Use WithoutCancel to preserve context values (like WorkDir)
		// while removing cancellation, so cleanup works after test ends
		cleanupCtx := context.WithoutCancel(ctx)
		if err := fs.RemoveAll(cleanupCtx, fsys, path); err != nil {
			t.Errorf("cleanup: RemoveAll(%q): %v", path, err)
		}
	})
}
