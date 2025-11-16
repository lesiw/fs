package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

// TestRemove tests removing files and empty directories.
func testRemove(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	rfs, ok := fsys.(fs.RemoveFS)
	if !ok {
		t.Skip("RemoveFS not supported")
	}

	// Test removing a file
	fileData := []byte("data")
	fileName := "test_remove_file.txt"
	if err := fs.WriteFile(ctx, fsys, fileName, fileData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write file: %v", err)
	}

	if err := rfs.Remove(ctx, fileName); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	// Verify file is gone
	_, statErr := fs.Stat(ctx, fsys, fileName)
	if statErr == nil {
		t.Errorf("stat after remove: file still exists")
	}

	// Test removing empty directory (skip if mkdir not supported)
	dirName := "test_remove_dir"
	mkdirErr := fs.Mkdir(ctx, fsys, dirName)
	if !errors.Is(mkdirErr, fs.ErrUnsupported) {
		if mkdirErr != nil {
			t.Fatalf("mkdir: %v", mkdirErr)
		}

		if err := rfs.Remove(ctx, dirName); err != nil {
			t.Fatalf("remove dir: %v", err)
		}

		// Verify directory is gone
		_, statErr = fs.Stat(ctx, fsys, dirName)
		if statErr == nil {
			t.Errorf("stat after remove: directory still exists")
		}

		// Test removing non-empty directory should fail
		nonemptyDir := "test_remove_nonempty"
		if err := fs.Mkdir(ctx, fsys, nonemptyDir); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		fileInDir := nonemptyDir + "/file.txt"
		writeErr := fs.WriteFile(ctx, fsys, fileInDir, fileData)
		if writeErr != nil {
			t.Fatalf("write: %v", writeErr)
		}

		removeErr := rfs.Remove(ctx, nonemptyDir)
		if removeErr == nil {
			t.Errorf("remove non-empty dir: expected error, got nil")
		}

		// Clean up
		if err := fs.RemoveAll(ctx, fsys, nonemptyDir); err != nil {
			t.Fatalf("cleanup: %v", err)
		}
	}
}

// TestRemoveAll tests recursive removal.
func testRemoveAll(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if RemoveAll is supported (either native or via fallback)
	_, hasRemoveAll := fsys.(fs.RemoveAllFS)
	_, hasRemove := fsys.(fs.RemoveFS)
	_, hasStat := fsys.(fs.StatFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)

	// Skip if neither native RemoveAllFS nor fallback requirements exist
	if !hasRemoveAll && (!hasRemove || !hasStat || !hasReadDir) {
		t.Skip(
			"RemoveAll not supported " +
				"(requires RemoveAllFS or RemoveFS+StatFS+ReadDirFS)",
		)
	}

	// Create nested structure
	testDir := "test_removeall"
	mkdirErr := fs.MkdirAll(ctx, fsys, testDir+"/a/b/c")
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for RemoveAll test)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdirall: %v", mkdirErr)
	}
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

	// Remove entire tree
	if err := fs.RemoveAll(ctx, fsys, testDir); err != nil {
		t.Fatalf("removeall: %v", err)
	}

	// Verify it's gone
	_, statErr := fs.Stat(ctx, fsys, testDir)
	if statErr == nil {
		t.Errorf("stat after removeall: directory still exists")
	}

	// Test RemoveAll on non-existent path should succeed
	nonexistentPath := "test_removeall_nonexistent"
	if err := fs.RemoveAll(ctx, fsys, nonexistentPath); err != nil {
		t.Errorf("removeall nonexistent: %v", err)
	}
}
