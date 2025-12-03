package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testRename(ctx context.Context, t *testing.T, fsys fs.FS) {
	rfs, ok := fsys.(fs.RenameFS)
	if !ok {
		t.Skip("RenameFS not supported")
	}

	t.Run("RenameFile", func(t *testing.T) {
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

		if err := rfs.Rename(ctx, oldName, newName); err != nil {
			t.Fatalf("rename file: %v", err)
		}

		// Verify old name is gone
		_, statErr := fs.Stat(ctx, fsys, oldName)
		if statErr == nil {
			t.Errorf("stat old name: file still exists")
		}

		// Verify new name exists
		info, statErr := fs.Stat(ctx, fsys, newName)
		if statErr != nil {
			t.Fatalf("stat new name: %v", statErr)
		}
		if info.IsDir() {
			t.Errorf("stat new name: expected file, got directory")
		}

		// Verify content is preserved
		data, readErr := fs.ReadFile(ctx, fsys, newName)
		if readErr != nil {
			t.Fatalf("read renamed file: %v", readErr)
		}
		if string(data) != "data" {
			t.Errorf("read renamed file: content changed")
		}
	})

	t.Run("RenameDir", func(t *testing.T) {
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

		if err := rfs.Rename(ctx, oldDirName, newDirName); err != nil {
			t.Fatalf("rename dir: %v", err)
		}

		// Verify directory was renamed
		info, statErr := fs.Stat(ctx, fsys, newDirName)
		if statErr != nil {
			t.Fatalf("stat renamed dir: %v", statErr)
		}
		if !info.IsDir() {
			t.Errorf("stat renamed dir: expected directory")
		}
	})
}
