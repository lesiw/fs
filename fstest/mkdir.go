package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testMkdir(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("Mkdir", func(t *testing.T) {
		testMkdirBasic(ctx, t, fsys)
	})

	t.Run("MkdirAll", func(t *testing.T) {
		testMkdirAll(ctx, t, fsys)
	})
}

func testMkdirBasic(ctx context.Context, t *testing.T, fsys fs.FS) {
	const dir = "test_mkdir"
	const file = dir + "/file.txt"

	_, hasMkdirFS := fsys.(fs.MkdirFS)

	err := fs.Mkdir(ctx, fsys, dir)
	if errors.Is(err, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported")
	}
	if err != nil {
		t.Fatalf("Mkdir(%q): %v", dir, err)
	}
	cleanup(ctx, t, fsys, dir)

	if hasMkdirFS {
		info, statErr := fs.Stat(ctx, fsys, dir)
		if statErr != nil {
			if !errors.Is(statErr, fs.ErrUnsupported) {
				t.Errorf("Stat(%q) after Mkdir: %v", dir, statErr)
			}
		} else if !info.IsDir() {
			t.Errorf("Stat(%q): IsDir() = false, want true", dir)
		}
	}

	testData := []byte("test content")
	writeErr := fs.WriteFile(ctx, fsys, file, testData)
	if writeErr != nil {
		if errors.Is(writeErr, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) after Mkdir: %v", file, writeErr)
	}

	data, err := fs.ReadFile(ctx, fsys, file)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", file, err)
	}
	if string(data) != string(testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", file, data, testData)
	}

	if hasMkdirFS {
		info, statErr := fs.Stat(ctx, fsys, dir)
		if statErr != nil {
			t.Errorf("Stat(%q) after creating file: %v", dir, statErr)
		} else if !info.IsDir() {
			t.Errorf("Stat(%q): IsDir() = false, want true", dir)
		}
	}

	err = fs.Mkdir(ctx, fsys, dir)
	if err == nil {
		// Some implementations may make Mkdir idempotent, which is acceptable
		t.Logf("Mkdir(%q) on existing directory succeeded (idempotent)", dir)
	}
}

func testMkdirAll(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	const path = "test_mkdirall/a/b/c"

	_, hasMkdirFS := fsys.(fs.MkdirFS)

	err := fs.MkdirAll(ctx, fsys, path)
	if errors.Is(err, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported")
	}
	if err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	cleanup(ctx, t, fsys, "test_mkdirall")

	if hasMkdirFS {
		info, statErr := fs.Stat(ctx, fsys, path)
		if statErr != nil {
			if !errors.Is(statErr, fs.ErrUnsupported) {
				t.Errorf("Stat(%q) after MkdirAll: %v", path, statErr)
			}
		} else if !info.IsDir() {
			t.Errorf("Stat(%q): IsDir() = false, want true", path)
		}
	}

	testFile := path + "/file.txt"
	testData := []byte("nested content")
	writeErr := fs.WriteFile(ctx, fsys, testFile, testData)
	if writeErr != nil {
		if errors.Is(writeErr, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) after MkdirAll: %v", testFile, writeErr)
	}

	data, err := fs.ReadFile(ctx, fsys, testFile)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", testFile, err)
	}
	if string(data) != string(testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", testFile, data, testData)
	}

	existingDir := "test_mkdirall/a/b"
	if err := fs.MkdirAll(ctx, fsys, existingDir); err != nil {
		t.Errorf("MkdirAll(%q) on existing directory: %v", existingDir, err)
	}
}
