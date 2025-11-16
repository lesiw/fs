package fstest

import (
	"context"
	"errors"
	"testing"
	"time"

	"lesiw.io/fs"
)

// TestChtimes tests changing file access and modification times.
func testChtimes(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Create test file
	fileName := "test_chtimes.txt"
	testData := []byte("chtimes test")
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

	// Set specific times
	atime := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	mtime := time.Date(2021, 6, 15, 18, 30, 0, 0, time.UTC)

	err := fs.Chtimes(ctx, fsys, fileName, atime, mtime)

	// ErrUnsupported is acceptable (capability not implemented)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("Chtimes not supported: %v", err)
			return
		}
		t.Fatalf("Chtimes(%q): %v", fileName, err)
	}

	// Verify modification time changed
	info, statErr := fs.Stat(ctx, fsys, fileName)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", fileName, statErr)
	}

	gotMtime := info.ModTime()
	// Allow 1 second tolerance for filesystem time granularity
	if gotMtime.Sub(mtime).Abs() > time.Second {
		t.Errorf(
			"Chtimes(%q): ModTime() = %v, want %v",
			fileName, gotMtime, mtime,
		)
	}

	// Verify file content unchanged
	data, readErr := fs.ReadFile(ctx, fsys, fileName)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) after chtimes: %v", fileName, readErr)
	}

	if string(data) != string(testData) {
		t.Errorf(
			"ReadFile(%q) after chtimes = %q, want %q",
			fileName, data, testData,
		)
	}
}
