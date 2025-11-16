package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

// TestSymlink tests creating and following symbolic links.
func testSymlink(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Create target file
	targetFile := "test_symlink_target.txt"
	testData := []byte("symlink target content")
	if err := fs.WriteFile(ctx, fsys, targetFile, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", targetFile, err)
	}
	t.Cleanup(func() {
		if err := fs.Remove(ctx, fsys, targetFile); err != nil {
			t.Errorf("Cleanup: Remove(%q): %v", targetFile, err)
		}
	})

	// Create symlink
	linkName := "test_symlink_link.txt"
	err := fs.Symlink(ctx, fsys, targetFile, linkName)

	// ErrUnsupported is acceptable (capability not implemented)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("Symlink not supported: %v", err)
			return
		}
		t.Fatalf("Symlink(%q, %q): %v", targetFile, linkName, err)
	}
	t.Cleanup(func() {
		if err := fs.Remove(ctx, fsys, linkName); err != nil {
			t.Errorf("Cleanup: Remove(%q): %v", linkName, err)
		}
	})

	// Read through symlink
	data, readErr := fs.ReadFile(ctx, fsys, linkName)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) through symlink: %v", linkName, readErr)
	}

	if string(data) != string(testData) {
		t.Errorf(
			"ReadFile(%q) through symlink = %q, want %q",
			linkName, data, testData,
		)
	}

	// Test symlink to directory
	targetDir := "test_symlink_dir"
	mkdirErr := fs.Mkdir(ctx, fsys, targetDir)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Logf("MkdirFS not supported, skipping directory symlink test")
		return
	}
	if mkdirErr != nil {
		t.Fatalf("Mkdir(%q): %v", targetDir, mkdirErr)
	}
	t.Cleanup(func() {
		if err := fs.Remove(ctx, fsys, targetDir); err != nil {
			t.Errorf("Cleanup: Remove(%q): %v", targetDir, err)
		}
	})

	dirLink := "test_symlink_dirlink"
	if err := fs.Symlink(ctx, fsys, targetDir, dirLink); err != nil {
		t.Fatalf("Symlink(%q, %q): %v", targetDir, dirLink, err)
	}
	t.Cleanup(func() {
		if err := fs.Remove(ctx, fsys, dirLink); err != nil {
			t.Errorf("Cleanup: Remove(%q): %v", dirLink, err)
		}
	})

	// Verify directory symlink
	info, statErr := fs.Stat(ctx, fsys, dirLink)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", dirLink, statErr)
	}

	if !info.IsDir() {
		t.Errorf(
			"Stat(%q) through symlink: IsDir() = false, want true",
			dirLink,
		)
	}
}

// TestReadlink tests reading symbolic link targets.
func testReadlink(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Create target
	targetFile := "test_readlink_target.txt"
	testData := []byte("target")
	if err := fs.WriteFile(ctx, fsys, targetFile, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", targetFile, err)
	}
	t.Cleanup(func() {
		if err := fs.Remove(ctx, fsys, targetFile); err != nil {
			t.Errorf("Cleanup: Remove(%q): %v", targetFile, err)
		}
	})

	// Create symlink (need to check if Symlink supported first)
	linkName := "test_readlink_link.txt"
	symlinkErr := fs.Symlink(ctx, fsys, targetFile, linkName)
	if symlinkErr != nil {
		if errors.Is(symlinkErr, fs.ErrUnsupported) {
			t.Logf(
				"Symlink not supported (required for Readlink test): %v",
				symlinkErr,
			)
			return
		}
		t.Fatalf("Symlink(%q, %q): %v", targetFile, linkName, symlinkErr)
	}
	t.Cleanup(func() {
		if err := fs.Remove(ctx, fsys, linkName); err != nil {
			t.Errorf("Cleanup: Remove(%q): %v", linkName, err)
		}
	})

	// Read symlink target
	target, err := fs.ReadLink(ctx, fsys, linkName)

	// ErrUnsupported is acceptable (capability not implemented)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("Readlink not supported: %v", err)
			return
		}
		t.Fatalf("Readlink(%q): %v", linkName, err)
	}

	if target != targetFile {
		t.Errorf("Readlink(%q) = %q, want %q", linkName, target, targetFile)
	}
}
