package fstest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"lesiw.io/fs"
)

// TestMkdir tests basic directory creation.
func testMkdir(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	const dir = "test_mkdir"
	const file = dir + "/file.txt"

	// Check if filesystem implements MkdirFS
	_, hasMkdirFS := fsys.(fs.MkdirFS)

	// Create a directory
	err := fs.Mkdir(ctx, fsys, dir)
	if errors.Is(err, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported")
	}
	if err != nil {
		t.Fatalf("Mkdir(%q): %v", dir, err)
	}
	cleanup(ctx, t, fsys, dir)

	// If filesystem implements MkdirFS, we can stat the empty directory
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

	// Verify directory works by creating a file in it
	testData := []byte("test content")
	writeErr := fs.WriteFile(ctx, fsys, file, testData)
	if writeErr != nil {
		if errors.Is(writeErr, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) after Mkdir: %v", file, writeErr)
	}

	// Verify file can be read
	data, err := fs.ReadFile(ctx, fsys, file)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", file, err)
	}
	if string(data) != string(testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", file, data, testData)
	}

	// For filesystems with MkdirFS, the directory should be visible after
	// creating a file in it
	if hasMkdirFS {
		info, statErr := fs.Stat(ctx, fsys, dir)
		if statErr != nil {
			t.Errorf("Stat(%q) after creating file: %v", dir, statErr)
		} else if !info.IsDir() {
			t.Errorf("Stat(%q): IsDir() = false, want true", dir)
		}
	}

	// Try to create directory again - should fail or be idempotent
	err = fs.Mkdir(ctx, fsys, dir)
	if err == nil {
		// Some implementations may make Mkdir idempotent, which is acceptable
		t.Logf("Mkdir(%q) on existing directory succeeded (idempotent)", dir)
	}
}

// TestMkdirAll tests creating nested directories.
func testMkdirAll(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	const path = "test_mkdirall/a/b/c"

	// Check if filesystem implements MkdirFS
	_, hasMkdirFS := fsys.(fs.MkdirFS)

	// Create nested directories
	err := fs.MkdirAll(ctx, fsys, path)
	if errors.Is(err, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported")
	}
	if err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	cleanup(ctx, t, fsys, "test_mkdirall")

	// If filesystem implements MkdirFS, we can stat the empty directory
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

	// Verify directories work by creating a file in the deepest one
	testFile := path + "/file.txt"
	testData := []byte("nested content")
	writeErr := fs.WriteFile(ctx, fsys, testFile, testData)
	if writeErr != nil {
		if errors.Is(writeErr, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) after MkdirAll: %v", testFile, writeErr)
	}

	// Verify file can be read
	data, err := fs.ReadFile(ctx, fsys, testFile)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", testFile, err)
	}
	if string(data) != string(testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", testFile, data, testData)
	}

	// MkdirAll on existing directory should succeed (idempotent)
	existingDir := "test_mkdirall/a/b"
	if err := fs.MkdirAll(ctx, fsys, existingDir); err != nil {
		t.Errorf("MkdirAll(%q) on existing directory: %v", existingDir, err)
	}
}

// TestReadDir tests reading directory contents.
func testReadDir(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	rdfs, ok := fsys.(fs.ReadDirFS)
	if !ok {
		t.Skip("ReadDirFS not supported")
	}

	const dir = "test_readdir"

	// Create test directory with contents
	mkdirErr := fs.Mkdir(ctx, fsys, dir)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestReadDir)")
	}
	if mkdirErr != nil {
		t.Fatalf("Mkdir(%q): %v", dir, mkdirErr)
	}
	cleanup(ctx, t, fsys, dir)

	// Create files and subdirectories
	file1Data := []byte("one")
	file1 := dir + "/file1.txt"
	if err := fs.WriteFile(ctx, fsys, file1, file1Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", file1, err)
	}
	file2Data := []byte("two")
	file2 := dir + "/file2.txt"
	if err := fs.WriteFile(ctx, fsys, file2, file2Data); err != nil {
		t.Fatalf("WriteFile(%q): %v", file2, err)
	}
	if err := fs.Mkdir(ctx, fsys, dir+"/subdir"); err != nil {
		t.Fatalf("Mkdir(%q): %v", dir+"/subdir", err)
	}
	// Create a file in subdir to make it visible in implementations
	// where directories are virtual (like S3)
	subdirFile := dir + "/subdir/nested.txt"
	subdirData := []byte("nested")
	writeErr := fs.WriteFile(ctx, fsys, subdirFile, subdirData)
	if writeErr != nil {
		t.Fatalf("WriteFile(%q): %v", subdirFile, writeErr)
	}

	// Read directory
	var names []string
	var entries []fs.DirEntry
	for entry, err := range rdfs.ReadDir(ctx, dir) {
		if err != nil {
			t.Fatalf("ReadDir(%q) iteration error: %v", dir, err)
		}
		names = append(names, entry.Name())
		entries = append(entries, entry)
	}

	// Check we got all entries
	if len(names) != 3 {
		t.Errorf("ReadDir(%q) returned %d entries, want 3", dir, len(names))
	}

	want := []string{"file1.txt", "file2.txt", "subdir"}
	slices.Sort(names)
	if !slices.Equal(names, want) {
		t.Errorf("ReadDir(%q) entries = %v, want %v", dir, names, want)
	}

	// Verify IsDir() is correct
	for _, entry := range entries {
		if entry.Name() == "subdir" && !entry.IsDir() {
			t.Errorf(
				"ReadDir(%q) entry %q: IsDir() = false, want true",
				dir, "subdir",
			)
		}
		if entry.Name() != "subdir" && entry.IsDir() {
			t.Errorf(
				"ReadDir(%q) entry %q: IsDir() = true, want false",
				dir, entry.Name(),
			)
		}
	}

	// Verify Size() is correct for files
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
			t.Errorf("entry.Info() for %q: %v", entry.Name(), infoErr)
			continue
		}

		if entry.Name() == "file1.txt" {
			if info.Size() != int64(len(file1Data)) {
				t.Errorf(
					"ReadDir(%q) entry %q: Size() = %d, want %d",
					dir, entry.Name(), info.Size(), len(file1Data),
				)
			}
		}
		if entry.Name() == "file2.txt" {
			if info.Size() != int64(len(file2Data)) {
				t.Errorf(
					"ReadDir(%q) entry %q: Size() = %d, want %d",
					dir, entry.Name(), info.Size(), len(file2Data),
				)
			}
		}
	}

	// Test ReadDir on a file (should return error or empty iterator)
	// ReadDir on a file path should fail - it's not a directory
	var fileReadCount int
	for _, err := range rdfs.ReadDir(ctx, file1) {
		if err == nil {
			fileReadCount++
		}
	}
	if got, want := fileReadCount, 0; got != want {
		t.Errorf("ReadDir(%q): got %d entries, want %d", file1, got, want)
	}
}
