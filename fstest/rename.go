package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testRename(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("RenameFile", func(t *testing.T) {
		testRenameFile(ctx, t, fsys)
	})
	t.Run("RenameDir", func(t *testing.T) {
		testRenameDir(ctx, t, fsys)
	})
}

func testRenameFile(ctx context.Context, t *testing.T, fsys fs.FS) {
	fileData := []byte("data")
	oldName := "test_rename_file_old.txt"
	newName := "test_rename_file_new.txt"
	if err := fs.WriteFile(ctx, fsys, oldName, fileData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write file: %v", err)
	}
	cleanup(ctx, t, fsys, newName)

	if err := fs.Rename(ctx, fsys, oldName, newName); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("RenameFS not supported")
		}
		t.Fatalf("rename file: %v", err)
	}

	_, statErr := fs.Stat(ctx, fsys, oldName)
	if statErr == nil {
		t.Errorf("stat old name: file still exists")
	}

	info, statErr := fs.Stat(ctx, fsys, newName)
	if statErr != nil {
		t.Fatalf("stat new name: %v", statErr)
	}
	if info.IsDir() {
		t.Errorf("stat new name: expected file, got directory")
	}

	data, readErr := fs.ReadFile(ctx, fsys, newName)
	if readErr != nil {
		t.Fatalf("read renamed file: %v", readErr)
	}
	if string(data) != "data" {
		t.Errorf("read renamed file: content changed")
	}
}

func testRenameDir(ctx context.Context, t *testing.T, fsys fs.FS) {
	oldDirName := "test_rename_dir_old"
	newDirName := "test_rename_dir_new"
	mkdirErr := fs.Mkdir(ctx, fsys, oldDirName)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdir: %v", mkdirErr)
	}
	cleanup(ctx, t, fsys, newDirName)

	if err := fs.Rename(ctx, fsys, oldDirName, newDirName); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("RenameFS not supported")
		}
		t.Fatalf("rename dir: %v", err)
	}

	info, statErr := fs.Stat(ctx, fsys, newDirName)
	if statErr != nil {
		t.Fatalf("stat renamed dir: %v", statErr)
	}
	if !info.IsDir() {
		t.Errorf("stat renamed dir: expected directory")
	}
}
