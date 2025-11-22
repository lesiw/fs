package fstest

import (
	"context"
	"errors"
	"io"
	"testing"

	"lesiw.io/fs"
)

// testWorkDir tests working directory context behavior.
func testWorkDir(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Create a test directory structure
	// /data/file.txt
	if err := fs.MkdirAll(ctx, fsys, "data"); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("mkdir not supported")
		}
		t.Fatalf("mkdir: %v", err)
	}
	cleanup(ctx, t, fsys, "data")

	testContent := []byte("hello from data")
	err := fs.WriteFile(ctx, fsys, "data/file.txt", testContent)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write file: %v", err)
	}

	// Now set working directory to data and try to read file.txt
	workCtx := fs.WithWorkDir(ctx, "data")
	rc, err := fs.Open(workCtx, fsys, "file.txt")
	if err != nil {
		t.Fatalf("open with working directory: %v", err)
	}

	buf, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read with working directory: %v", err)
	}

	if string(buf) != string(testContent) {
		t.Errorf(
			"working directory read: got %q, want %q",
			string(buf), string(testContent),
		)
	}
}
