package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

// TestStat tests file metadata retrieval.
func testStat(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	sfs, ok := fsys.(fs.StatFS)
	if !ok {
		t.Skip("StatFS not supported")
	}

	// Test stat on file
	fileName := "test_stat_file.txt"
	helloData := []byte("hello")
	if err := fs.WriteFile(ctx, fsys, fileName, helloData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write file: %v", err)
	}

	info, statErr := sfs.Stat(ctx, fileName)
	if statErr != nil {
		t.Fatalf("stat file: %v", statErr)
	}

	if info.IsDir() {
		t.Errorf("stat file: IsDir() = true, expected false")
	}

	if info.Name() != fileName {
		t.Errorf("stat file: Name() = %q, expected %q", info.Name(), fileName)
	}

	if info.Size() != 5 {
		t.Errorf("stat file: Size() = %d, expected 5", info.Size())
	}

	// Test stat on directory (skip if mkdir not supported)
	dirName := "test_stat_dir"
	mkdirErr := fs.Mkdir(ctx, fsys, dirName)
	if !errors.Is(mkdirErr, fs.ErrUnsupported) {
		if mkdirErr != nil {
			t.Fatalf("mkdir: %v", mkdirErr)
		}

		// Create a file in the directory to make it visible
		// in implementations where directories are virtual (like S3)
		dirFile := dirName + "/file.txt"
		dirData := []byte("test")
		if err := fs.WriteFile(ctx, fsys, dirFile, dirData); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("WriteFile(%q): %v", dirFile, err)
		}

		info, statErr = sfs.Stat(ctx, dirName)
		if statErr != nil {
			t.Fatalf("stat dir: %v", statErr)
		}

		if !info.IsDir() {
			t.Errorf("stat dir: IsDir() = false, expected true")
		}

		if info.Name() != dirName {
			t.Errorf(
				"stat dir: Name() = %q, expected %q", info.Name(), dirName,
			)
		}
	}

	// Test stat on non-existent file
	nonexistent := "test_stat_nonexistent"
	_, statErr = sfs.Stat(ctx, nonexistent)
	if statErr == nil {
		t.Errorf("stat nonexistent: expected error, got nil")
	}

	// Clean up
	if err := fs.Remove(ctx, fsys, fileName); err != nil {
		t.Fatalf("cleanup file: %v", err)
	}
	if err := fs.RemoveAll(ctx, fsys, dirName); err != nil {
		t.Fatalf("cleanup dir: %v", err)
	}
}
