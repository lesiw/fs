package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

// TestChmod tests changing file permissions.
func testChmod(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Create test file
	fileName := "test_chmod.txt"
	testData := []byte("chmod test")
	if err := fs.WriteFile(ctx, fsys, fileName, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", fileName, err)
	}
	t.Cleanup(func() {
		if err := fs.Remove(ctx, fsys, fileName); err != nil {
			t.Errorf("Cleanup: Remove(%q): %v", fileName, err)
		}
	})

	// Change permissions (via helper function which uses fallback if needed)
	newMode := fs.Mode(0600)
	err := fs.Chmod(ctx, fsys, fileName, newMode)

	// ErrUnsupported is acceptable (capability not implemented)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("Chmod not supported: %v", err)
			return
		}
		t.Fatalf("Chmod(%q, %o): %v", fileName, newMode, err)
	}

	// Verify new permissions
	info, statErr := fs.Stat(ctx, fsys, fileName)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", fileName, statErr)
	}

	gotMode := info.Mode() & 0777
	if gotMode != newMode {
		t.Errorf(
			"Chmod(%q, %o): Mode() = %o, want %o",
			fileName, newMode, gotMode, newMode,
		)
	}

	// Test chmod on directory
	dirName := "test_chmod_dir"
	mkdirErr := fs.Mkdir(ctx, fsys, dirName)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Logf("MkdirFS not supported, skipping directory chmod test")
		return
	}
	if mkdirErr != nil {
		t.Fatalf("Mkdir(%q): %v", dirName, mkdirErr)
	}
	t.Cleanup(func() {
		if err := fs.Remove(ctx, fsys, dirName); err != nil {
			t.Errorf("Cleanup: Remove(%q): %v", dirName, err)
		}
	})

	dirMode := fs.Mode(0700)
	if err := fs.Chmod(ctx, fsys, dirName, dirMode); err != nil {
		t.Fatalf("Chmod(%q, %o): %v", dirName, dirMode, err)
	}

	info, statErr = fs.Stat(ctx, fsys, dirName)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", dirName, statErr)
	}

	gotMode = info.Mode() & 0777
	if gotMode != dirMode {
		t.Errorf(
			"Chmod(%q, %o): Mode() = %o, want %o",
			dirName, dirMode, gotMode, dirMode,
		)
	}
}
