package fstest

import (
	"context"
	"testing"

	"lesiw.io/fs"
)

func testLocalize(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("Localize", func(t *testing.T) {
		testLocalizeIdempotent(ctx, t, fsys)
	})
}

func testLocalizeIdempotent(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
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
			localized1, err := fs.Localize(ctx, fsys, tt.path)
			if err != nil {
				t.Skipf("Localize(%q) failed: %v", tt.path, err)
			}

			// Second localization (idempotency test)
			localized2, err := fs.Localize(ctx, fsys, localized1)
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
