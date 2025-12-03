package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testAppend(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("Append", func(t *testing.T) {
		testAppendToFile(ctx, t, fsys)
	})
}

func testAppendToFile(ctx context.Context, t *testing.T, fsys fs.FS) {
	name := "test_append.txt"

	initialData := []byte("initial")
	if err := fs.WriteFile(ctx, fsys, name, initialData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	f, err := fs.Append(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Append not supported")
		}
		t.Fatalf("Append(%q): %v", name, err)
	}

	if _, writeErr := f.Write([]byte(" appended")); writeErr != nil {
		f.Close()
		t.Fatalf("Write(%q): %v", " appended", writeErr)
	}

	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}

	data, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	want := "initial appended"
	if string(data) != want {
		t.Errorf(
			"ReadFile(%q) after Append = %q, want %q", name, data, want,
		)
	}
}
