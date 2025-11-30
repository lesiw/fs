package fstest

import (
	"context"
	"testing"

	"lesiw.io/fs"
)

// TestLocalize tests that LocalizeFS.Localize is idempotent.
//
// Skips if the filesystem does not implement LocalizeFS.
func TestLocalize(ctx context.Context, t *testing.T, fsys fs.FS) {
	lfs, ok := fsys.(fs.LocalizeFS)
	if !ok {
		t.Skip("filesystem does not implement LocalizeFS")
	}

	tests := []struct {
		name string
		path string
	}{
		{"simple file", "test.txt"},
		{"nested path", "dir/subdir/file.txt"},
		{"with dots", "dir/../other/file.txt"},
		{"root", "."},
		{"directory", "dir"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First localization
			localized1, err := lfs.Localize(ctx, tt.path)
			if err != nil {
				t.Skipf("Localize(%q) failed: %v", tt.path, err)
			}

			// Second localization (idempotency test)
			localized2, err := lfs.Localize(ctx, localized1)
			if err != nil {
				t.Errorf("Localize(Localize(%q)) failed: %v", tt.path, err)
				return
			}

			// Should be equal
			if localized1 != localized2 {
				t.Errorf(
					"Localize not idempotent for %q:\n"+
						"  first:  %q\n  second: %q",
					tt.path, localized1, localized2,
				)
			}
		})
	}
}
