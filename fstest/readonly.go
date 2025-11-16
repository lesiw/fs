package fstest

import (
	"context"
	"io"
	"testing"

	"lesiw.io/fs"
)

// testReadOnly runs tests appropriate for read-only filesystems with
// pre-populated files specified in expected.
//
// Similar to testing/fstest.TestFS, this validates that:
// - All expected files exist and are readable
// - Multiple reads return consistent data
// - Stat info matches actual file content (if StatFS supported)
// - Basic file operations work correctly
func testReadOnly(
	ctx context.Context, t *testing.T, fsys fs.FS, expected []string,
) {
	t.Helper()

	if len(expected) == 0 {
		t.Fatal("testReadOnly requires expected files")
	}

	// Test that all expected files exist and are readable
	t.Run("ExpectedFiles", func(t *testing.T) {
		for _, path := range expected {
			t.Run(path, func(t *testing.T) {
				// First read - Open and ReadAll
				f, err := fsys.Open(ctx, path)
				if err != nil {
					t.Fatalf("Open(%q): %v", path, err)
				}
				data1, err := io.ReadAll(f)
				if err != nil {
					f.Close()
					t.Fatalf("ReadAll(%q): %v", path, err)
				}
				if closeErr := f.Close(); closeErr != nil {
					t.Errorf("Close(%q): %v", path, closeErr)
				}

				// Second read - verify consistency
				f, err = fsys.Open(ctx, path)
				if err != nil {
					t.Fatalf("second Open(%q): %v", path, err)
				}
				data2, err := io.ReadAll(f)
				if err != nil {
					f.Close()
					t.Fatalf("second ReadAll(%q): %v", path, err)
				}
				f.Close()

				// Verify multiple reads return same data
				if string(data1) != string(data2) {
					msg := "Open+ReadAll(%q): inconsistent data:\n" +
						"first:  %q\nsecond: %q"
					t.Errorf(msg, path, data1, data2)
				}

				// Test Stat if supported
				if sfs, ok := fsys.(fs.StatFS); ok {
					info, err := sfs.Stat(ctx, path)
					if err != nil {
						t.Errorf("Stat(%q): %v", path, err)
					} else {
						// Verify stat info matches what we read
						size := info.Size()
						readSize := int64(len(data1))
						if !info.IsDir() && size != readSize {
							t.Errorf(
								"Stat(%q).Size() = %d, but Read got %d bytes",
								path, size, readSize,
							)
						}
					}
				}
			})
		}
	})
}
