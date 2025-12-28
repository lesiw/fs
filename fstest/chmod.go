package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testChmod(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("ChmodFile", func(t *testing.T) {
		testChmodFile(ctx, t, fsys)
	})
	t.Run("ChmodDir", func(t *testing.T) {
		testChmodDir(ctx, t, fsys)
	})
}

func testChmodFile(ctx context.Context, t *testing.T, fsys fs.FS) {
	fileName := "test_chmod_file.txt"
	testData := []byte("chmod test")
	if err := fs.WriteFile(ctx, fsys, fileName, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", fileName, err)
	}
	cleanup(ctx, t, fsys, fileName)

	newMode := fs.Mode(0600)
	err := fs.Chmod(ctx, fsys, fileName, newMode)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Chmod not supported")
		}
		t.Fatalf("Chmod(%q, %o): %v", fileName, newMode, err)
	}

	info, statErr := fs.Stat(ctx, fsys, fileName)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrUnsupported) {
			t.Skip("Stat not supported")
		}
		t.Fatalf("Stat(%q): %v", fileName, statErr)
	}

	gotMode := info.Mode() & 0777
	if gotMode != newMode {
		t.Errorf(
			"Chmod(%q, %o): Mode() = %o, want %o",
			fileName, newMode, gotMode, newMode,
		)
	}
}

func testChmodDir(ctx context.Context, t *testing.T, fsys fs.FS) {
	dirName := "test_chmod_dir"
	mkdirErr := fs.Mkdir(ctx, fsys, dirName)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported")
	}
	if mkdirErr != nil {
		t.Fatalf("Mkdir(%q): %v", dirName, mkdirErr)
	}
	cleanup(ctx, t, fsys, dirName)

	dirMode := fs.Mode(0700)
	if err := fs.Chmod(ctx, fsys, dirName, dirMode); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Chmod not supported")
		}
		t.Fatalf("Chmod(%q, %o): %v", dirName, dirMode, err)
	}

	info, statErr := fs.Stat(ctx, fsys, dirName)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrUnsupported) {
			t.Skip("Stat not supported")
		}
		t.Fatalf("Stat(%q): %v", dirName, statErr)
	}

	gotMode := info.Mode() & 0777
	if gotMode != dirMode {
		t.Errorf(
			"Chmod(%q, %o): Mode() = %o, want %o",
			dirName, dirMode, gotMode, dirMode,
		)
	}
}
