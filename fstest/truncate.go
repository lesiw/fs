package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

// TestTruncate tests truncating files to a specific size.
func testTruncate(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Create test file with initial content
	fileName := "test_truncate.txt"
	testData := []byte("hello world this is a test")
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

	// Truncate to smaller size
	newSize := int64(5)
	err := fs.Truncate(ctx, fsys, fileName, newSize)

	// ErrUnsupported is acceptable (capability not implemented)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("Truncate not supported: %v", err)
			return
		}
		t.Fatalf("Truncate(%q, %d): %v", fileName, newSize, err)
	}

	// Verify new size
	info, statErr := fs.Stat(ctx, fsys, fileName)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", fileName, statErr)
	}

	if info.Size() != newSize {
		t.Errorf(
			"Truncate(%q, %d): Size() = %d, want %d",
			fileName, newSize, info.Size(), newSize,
		)
	}

	// Verify content
	data, readErr := fs.ReadFile(ctx, fsys, fileName)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) after truncate: %v", fileName, readErr)
	}

	expected := testData[:newSize]
	if string(data) != string(expected) {
		t.Errorf(
			"ReadFile(%q) after truncate = %q, want %q",
			fileName, data, expected,
		)
	}

	// Truncate to larger size (should extend with zeros)
	largerSize := int64(10)
	if err := fs.Truncate(ctx, fsys, fileName, largerSize); err != nil {
		t.Fatalf("Truncate(%q, %d): %v", fileName, largerSize, err)
	}

	info, statErr = fs.Stat(ctx, fsys, fileName)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", fileName, statErr)
	}

	if info.Size() != largerSize {
		t.Errorf(
			"Truncate(%q, %d): Size() = %d, want %d",
			fileName, largerSize, info.Size(), largerSize,
		)
	}
}
