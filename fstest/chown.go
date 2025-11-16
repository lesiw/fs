package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

// TestChown tests changing file ownership.
// Note: This test requires appropriate permissions and may skip on some
// systems.
func testChown(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Create test file
	fileName := "test_chown.txt"
	testData := []byte("chown test")
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

	// Get current ownership
	_, statErr := fs.Stat(ctx, fsys, fileName)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", fileName, statErr)
	}

	// Try to chown to same user/group (should always work)
	// Most systems don't allow non-root users to change ownership to
	// different users, so we use -1, -1 which means "don't change"
	err := fs.Chown(ctx, fsys, fileName, -1, -1)

	// ErrUnsupported is acceptable (capability not implemented)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("Chown not supported: %v", err)
			return
		}
		t.Fatalf("Chown(%q, -1, -1): %v", fileName, err)
	}

	// Verify file still exists and is readable
	data, readErr := fs.ReadFile(ctx, fsys, fileName)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) after chown: %v", fileName, readErr)
	}

	if string(data) != string(testData) {
		t.Errorf(
			"ReadFile(%q) after chown = %q, want %q",
			fileName, data, testData,
		)
	}
}
