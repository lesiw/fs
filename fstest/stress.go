package fstest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"lesiw.io/fs"
)

func testStress(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("MixedOperations", func(t *testing.T) {
		testMixedOperations(ctx, t, fsys)
	})

	t.Run("ConcurrentReads", func(t *testing.T) {
		testConcurrentReads(ctx, t, fsys)
	})

	t.Run("ModifyAndRead", func(t *testing.T) {
		testModifyAndRead(ctx, t, fsys)
	})
}

// TestMixedOperations performs a stress test that combines multiple filesystem
// operations in realistic patterns. This tests that implementations correctly
// handle complex workflows.
func testMixedOperations(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	// Create a directory structure with files at various levels
	baseDir := "stress_test"
	mkdirErr := fs.MkdirAll(ctx, fsys, baseDir+"/a/b/c")
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestMixedOperations)")
	}
	if mkdirErr != nil {
		t.Fatalf("MkdirAll(): %v", mkdirErr)
	}
	cleanup(ctx, t, fsys, baseDir)
	if err := fs.MkdirAll(ctx, fsys, baseDir+"/d/e"); err != nil {
		t.Fatalf("MkdirAll(): %v", err)
	}

	// Write files at various levels
	testFiles := map[string][]byte{
		baseDir + "/root.txt":         []byte("root level"),
		baseDir + "/a/level1.txt":     []byte("level 1"),
		baseDir + "/a/b/level2.txt":   []byte("level 2"),
		baseDir + "/a/b/c/level3.txt": []byte("level 3"),
		baseDir + "/d/other.txt":      []byte("other branch"),
	}

	for path, content := range testFiles {
		if err := fs.WriteFile(ctx, fsys, path, content); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("WriteFile(%q): %v", path, err)
		}
	}

	// Verify all files via Walk
	walkCount := 0
	for entry, walkErr := range fs.Walk(ctx, fsys, baseDir, -1) {
		if walkErr != nil {
			t.Errorf("Walk() error: %v", walkErr)
			continue
		}
		if !entry.IsDir() {
			walkCount++
			// Verify file content
			data, readErr := fs.ReadFile(ctx, fsys, entry.Path())
			if readErr != nil {
				t.Errorf("ReadFile(%q): %v", entry.Path(), readErr)
				continue
			}
			expected, ok := testFiles[entry.Path()]
			if !ok {
				t.Errorf("unexpected file in Walk: %q", entry.Path())
				continue
			}
			if !bytes.Equal(data, expected) {
				t.Errorf(
					"file %q content = %q, want %q",
					entry.Path(), data, expected,
				)
			}
		}
	}

	if walkCount != len(testFiles) {
		t.Errorf("Walk() found %d files, want %d", walkCount, len(testFiles))
	}

	// Rename a file and verify
	oldPath := baseDir + "/a/level1.txt"
	newPath := baseDir + "/a/renamed.txt"
	if err := fs.Rename(ctx, fsys, oldPath, newPath); err != nil {
		if !errors.Is(err, fs.ErrUnsupported) {
			t.Errorf("Rename(%q, %q): %v", oldPath, newPath, err)
		}
	} else {
		// Verify old path is gone
		if _, err := fs.Stat(ctx, fsys, oldPath); err == nil {
			t.Errorf(
				"Stat(%q) after rename succeeded, want error",
				oldPath,
			)
		}
		// Verify new path exists
		data, err := fs.ReadFile(ctx, fsys, newPath)
		if err != nil {
			t.Errorf("ReadFile(%q) after rename: %v", newPath, err)
		} else if !bytes.Equal(data, testFiles[oldPath]) {
			t.Errorf(
				"renamed file content = %q, want %q",
				data, testFiles[oldPath],
			)
		}
	}

	// Use Glob to find specific files
	pattern := baseDir + "/*/*.txt"
	matches, globErr := fs.Glob(ctx, fsys, pattern)
	if globErr != nil {
		t.Errorf("Glob(%q): %v", pattern, globErr)
	} else if len(matches) < 2 {
		t.Errorf(
			"Glob(%q) found %d matches, want at least 2",
			pattern, len(matches),
		)
	}

	// Remove a specific file
	removeFile := baseDir + "/d/other.txt"
	if err := fs.Remove(ctx, fsys, removeFile); err != nil {
		t.Errorf("Remove(%q): %v", removeFile, err)
	} else {
		// Verify file is gone
		if _, err := fs.Stat(ctx, fsys, removeFile); err == nil {
			t.Errorf("Stat(%q) after remove succeeded, want error", removeFile)
		}
	}
}

// TestConcurrentReads tests that multiple concurrent read operations work
// correctly. This is a basic concurrency test that doesn't use goroutines
// but does test that file handles don't interfere with each other.
func testConcurrentReads(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	// Create test files
	const numFiles = 5
	testDir := "concurrent_reads"
	mkdirErr := fs.Mkdir(ctx, fsys, testDir)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestConcurrentReads)")
	}
	if mkdirErr != nil {
		t.Fatalf("Mkdir(%q): %v", testDir, mkdirErr)
	}
	cleanup(ctx, t, fsys, testDir)

	type fileInfo struct {
		path    string
		content []byte
		reader  io.ReadCloser
	}

	testFiles := make([]fileInfo, numFiles)
	for i := 0; i < numFiles; i++ {
		path := fmt.Sprintf("%s/file%d.txt", testDir, i)
		content := []byte(fmt.Sprintf("content %d", i))
		testFiles[i] = fileInfo{path: path, content: content}

		if err := fs.WriteFile(ctx, fsys, path, content); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("WriteFile(%q): %v", path, err)
		}
	}

	// Open all files simultaneously (without closing)
	for i := range testFiles {
		r, err := fs.Open(ctx, fsys, testFiles[i].path)
		if err != nil {
			t.Fatalf("Open(%q): %v", testFiles[i].path, err)
		}
		testFiles[i].reader = r
	}

	// Read from all files
	for i := range testFiles {
		data := make([]byte, len(testFiles[i].content))
		n, readErr := testFiles[i].reader.Read(data)
		if readErr != nil {
			t.Errorf("Read() from %q: %v", testFiles[i].path, readErr)
		}
		if n != len(testFiles[i].content) {
			t.Errorf(
				"Read() from %q = %d bytes, want %d",
				testFiles[i].path, n, len(testFiles[i].content),
			)
		}
		if !bytes.Equal(data, testFiles[i].content) {
			t.Errorf(
				"Read() from %q = %q, want %q",
				testFiles[i].path, data, testFiles[i].content,
			)
		}
	}

	// Close all readers
	for i := range testFiles {
		if err := testFiles[i].reader.Close(); err != nil {
			t.Errorf("Close() reader %d: %v", i, err)
		}
	}
}

// TestModifyAndRead tests a realistic workflow of creating, modifying, and
// reading files in various ways.
func testModifyAndRead(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	testDir := "modify_test"
	mkdirErr := fs.Mkdir(ctx, fsys, testDir)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestModifyAndRead)")
	}
	if mkdirErr != nil {
		t.Fatalf("Mkdir(%q): %v", testDir, mkdirErr)
	}
	cleanup(ctx, t, fsys, testDir)

	filePath := testDir + "/data.txt"

	// Initial write
	initial := []byte("initial content")
	if err := fs.WriteFile(ctx, fsys, filePath, initial); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) initial: %v", filePath, err)
	}

	// Verify initial content
	data, err := fs.ReadFile(ctx, fsys, filePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) after initial write: %v", filePath, err)
	}
	if !bytes.Equal(data, initial) {
		t.Errorf(
			"initial ReadFile(%q) = %q, want %q",
			filePath, data, initial,
		)
	}

	// Overwrite with shorter content
	shorter := []byte("short")
	writeErr := fs.WriteFile(ctx, fsys, filePath, shorter)
	if writeErr != nil {
		if errors.Is(writeErr, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) overwrite: %v", filePath, writeErr)
	}

	// Verify overwrite truncated properly
	data, err = fs.ReadFile(ctx, fsys, filePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) after overwrite: %v", filePath, err)
	}
	if !bytes.Equal(data, shorter) {
		t.Errorf(
			"overwrite ReadFile(%q) = %q, want %q",
			filePath, data, shorter,
		)
	}

	// Append to file
	f, err := fs.Append(ctx, fsys, filePath)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("Append not supported, skipping append portion of test")
			return
		}
		t.Fatalf("Append(%q): %v", filePath, err)
	}

	appended := []byte(" appended")
	_, writeErr = f.Write(appended)
	if writeErr != nil {
		f.Close()
		t.Fatalf("Write() append: %v", writeErr)
	}

	closeErr := f.Close()
	if closeErr != nil {
		t.Fatalf("Close() after append: %v", closeErr)
	}

	// Verify append worked
	data, err = fs.ReadFile(ctx, fsys, filePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) after append: %v", filePath, err)
	}

	expected := append(shorter, appended...)
	if !bytes.Equal(data, expected) {
		t.Errorf(
			"append ReadFile(%q) = %q, want %q",
			filePath, data, expected,
		)
	}
}
