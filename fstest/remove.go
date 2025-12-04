package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testRemove(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("RemoveFile", func(t *testing.T) {
		fileData := []byte("data")
		fileName := "test_remove_file.txt"
		if err := fs.WriteFile(ctx, fsys, fileName, fileData); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("write file: %v", err)
		}
		cleanup(ctx, t, fsys, fileName)

		if err := fs.Remove(ctx, fsys, fileName); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("RemoveFS not supported")
			}
			t.Fatalf("remove file: %v", err)
		}

		_, statErr := fs.Stat(ctx, fsys, fileName)
		if statErr == nil {
			t.Errorf("stat after remove: file still exists")
		}
	})

	t.Run("RemoveDir", func(t *testing.T) {
		dirName := "test_remove_dir"
		mkdirErr := fs.Mkdir(ctx, fsys, dirName)
		if errors.Is(mkdirErr, fs.ErrUnsupported) {
			t.Skip("MkdirFS not supported")
		}
		if mkdirErr != nil {
			t.Fatalf("mkdir: %v", mkdirErr)
		}
		cleanup(ctx, t, fsys, dirName)

		if err := fs.Remove(ctx, fsys, dirName); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("RemoveFS not supported")
			}
			t.Fatalf("remove dir: %v", err)
		}

		_, statErr := fs.Stat(ctx, fsys, dirName)
		if statErr == nil {
			t.Errorf("stat after remove: directory still exists")
		}
	})

	t.Run("RemoveNonempty", func(t *testing.T) {
		mkdirErr := fs.Mkdir(ctx, fsys, "")
		if errors.Is(mkdirErr, fs.ErrUnsupported) {
			t.Skip("MkdirFS not supported")
		}

		fileData := []byte("data")
		nonemptyDir := "test_remove_nonempty"
		if err := fs.Mkdir(ctx, fsys, nonemptyDir); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		cleanup(ctx, t, fsys, nonemptyDir)

		fileInDir := nonemptyDir + "/file.txt"
		writeErr := fs.WriteFile(ctx, fsys, fileInDir, fileData)
		if writeErr != nil {
			t.Fatalf("write: %v", writeErr)
		}

		removeErr := fs.Remove(ctx, fsys, nonemptyDir)
		if removeErr == nil {
			t.Errorf("remove non-empty dir: expected error, got nil")
		} else if errors.Is(removeErr, fs.ErrUnsupported) {
			t.Skip("RemoveFS not supported")
		}
	})
}

func testRemoveAll(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	_, hasRemoveAll := fsys.(fs.RemoveAllFS)
	_, hasRemove := fsys.(fs.RemoveFS)
	_, hasStat := fsys.(fs.StatFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)

	if !hasRemoveAll && (!hasRemove || !hasStat || !hasReadDir) {
		t.Skip(
			"RemoveAll not supported " +
				"(requires RemoveAllFS or RemoveFS+StatFS+ReadDirFS)",
		)
	}

	testDir := "test_removeall"
	mkdirErr := fs.MkdirAll(ctx, fsys, testDir+"/a/b/c")
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for RemoveAll test)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdirall: %v", mkdirErr)
	}
	cleanup(ctx, t, fsys, testDir)

	file1Data := []byte("one")
	file1 := testDir + "/file1.txt"
	if err := fs.WriteFile(ctx, fsys, file1, file1Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write file1: %v", err)
	}
	file2Data := []byte("two")
	file2 := testDir + "/a/file2.txt"
	if err := fs.WriteFile(ctx, fsys, file2, file2Data); err != nil {
		t.Fatalf("write file2: %v", err)
	}
	file3Data := []byte("three")
	file3 := testDir + "/a/b/c/file3.txt"
	if err := fs.WriteFile(ctx, fsys, file3, file3Data); err != nil {
		t.Fatalf("write file3: %v", err)
	}

	if err := fs.RemoveAll(ctx, fsys, testDir); err != nil {
		t.Fatalf("removeall: %v", err)
	}

	_, statErr := fs.Stat(ctx, fsys, testDir)
	if statErr == nil {
		t.Errorf("stat after removeall: directory still exists")
	}

	nonexistentPath := "test_removeall_nonexistent"
	if err := fs.RemoveAll(ctx, fsys, nonexistentPath); err != nil {
		t.Errorf("removeall nonexistent: %v", err)
	}
}
