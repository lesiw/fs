package fstest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lesiw.io/fs"
)

// TestTempDir tests creating temporary directories.
func testTempDir(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if TempDir is supported (either native or via fallback)
	_, hasTemp := fsys.(fs.TempFS)
	_, hasMkdir := fsys.(fs.MkdirFS)

	// Skip if neither native TempFS nor fallback requirement
	// (MkdirFS) is present
	if !hasTemp && !hasMkdir {
		t.Skip("TempDir not supported (requires TempFS or MkdirFS)")
	}

	// Create temp directory with prefix
	prefix := "test_tempdir"
	tempDir, err := fs.TempDir(ctx, fsys, prefix)

	// ErrUnsupported is acceptable (capability not implemented)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("TempDir not supported: %v", err)
			return
		}
		t.Fatalf("TempDir(%q): %v", prefix, err)
	}
	t.Cleanup(func() {
		if removeErr := fs.RemoveAll(ctx, fsys, tempDir); removeErr != nil {
			t.Errorf("Cleanup: RemoveAll(%q): %v", tempDir, removeErr)
		}
	})

	// Verify name contains prefix
	if !strings.Contains(tempDir, prefix) {
		t.Errorf(
			"TempDir(%q): path = %q, want to contain %q",
			prefix, tempDir, prefix,
		)
	}

	// Create file in temp directory to make it visible in implementations
	// where directories are virtual (like S3)
	testFile := tempDir + "/test.txt"
	testData := []byte("temp file content")
	writeErr := fs.WriteFile(ctx, fsys, testFile, testData)
	if writeErr != nil {
		if errors.Is(writeErr, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", testFile, writeErr)
	}

	// Verify temp directory exists after creating file in it
	info, statErr := fs.Stat(ctx, fsys, tempDir)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", tempDir, statErr)
	}

	if !info.IsDir() {
		t.Errorf("TempDir(%q): IsDir() = false, want true", prefix)
	}

	// Verify file was created
	data, readErr := fs.ReadFile(ctx, fsys, testFile)
	if readErr != nil {
		t.Fatalf("ReadFile(%q): %v", testFile, readErr)
	}

	if string(data) != string(testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", testFile, data, testData)
	}

	// Create another temp dir to verify uniqueness
	tempDir2, err2 := fs.TempDir(ctx, fsys, prefix)
	if err2 != nil {
		t.Fatalf("TempDir(%q) second call: %v", prefix, err2)
	}
	t.Cleanup(func() {
		if removeErr := fs.RemoveAll(ctx, fsys, tempDir2); removeErr != nil {
			t.Errorf("Cleanup: RemoveAll(%q): %v", tempDir2, removeErr)
		}
	})

	if tempDir == tempDir2 {
		t.Errorf(
			"TempDir(%q) created duplicate names: %q",
			prefix, tempDir,
		)
	}
}
