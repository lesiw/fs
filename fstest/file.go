package fstest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"lesiw.io/fs"
)

// TestCreateAndRead tests basic file creation and reading.
func testCreateAndRead(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	const name = "test_create.txt"
	testData := []byte("hello world")

	// Create a new file
	f, err := fs.Create(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("Create(%q): %v", name, err)
	}

	// Write data
	n, err := f.Write(testData)
	if err != nil {
		f.Close()
		t.Fatalf("Write(%q): %v", testData, err)
	}
	if n != len(testData) {
		f.Close()
		t.Fatalf("Write(%q) = %d bytes, want %d", testData, n, len(testData))
	}

	// Close the file
	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}

	// Open and read back
	// (wrapped in func to ensure defer executes before Remove)
	func() {
		r, err := fs.Open(ctx, fsys, name)
		if err != nil {
			t.Fatalf("Open(%q): %v", name, err)
		}
		defer r.Close()

		readData, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll(): %v", err)
		}

		if !bytes.Equal(readData, testData) {
			t.Errorf("ReadAll() = %q, want %q", readData, testData)
		}
	}()

	// Clean up
	if err := fs.Remove(ctx, fsys, name); err != nil {
		t.Fatalf("Remove(%q): %v", name, err)
	}
}

// TestWriteFile tests the WriteFile helper function.
func testWriteFile(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	const name = "test_write.txt"
	testData := []byte("test data for writefile")

	// Write file
	if err := fs.WriteFile(ctx, fsys, name, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}

	// Read back
	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	if !bytes.Equal(readData, testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", name, readData, testData)
	}

	// Test overwrite (WriteFile should truncate)
	newData := []byte("new")
	writeErr := fs.WriteFile(ctx, fsys, name, newData)
	if writeErr != nil {
		t.Fatalf("WriteFile(%q) overwrite: %v", name, writeErr)
	}

	readData, err = fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q) after overwrite: %v", name, err)
	}

	if !bytes.Equal(readData, newData) {
		t.Errorf(
			"ReadFile(%q) after overwrite = %q, want %q",
			name, readData, newData,
		)
	}

	// Clean up
	if err := fs.Remove(ctx, fsys, name); err != nil {
		t.Fatalf("Remove(%q): %v", name, err)
	}
}

// TestCreateTruncates tests that Create truncates existing files.
func testCreateTruncates(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	const name = "test_truncate.txt"

	// Create file with initial data
	origData := []byte("original data")
	if err := fs.WriteFile(ctx, fsys, name, origData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}

	// Truncate via Create (which truncates existing files)
	f, err := fs.Create(ctx, fsys, name)
	if err != nil {
		t.Fatalf("Create(%q) for truncate: %v", name, err)
	}
	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}

	// File should be empty now
	data, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q) after truncate: %v", name, err)
	}
	if len(data) != 0 {
		t.Errorf(
			"ReadFile(%q) after truncate = %d bytes, want 0",
			name, len(data),
		)
	}

	// Clean up
	if err := fs.Remove(ctx, fsys, name); err != nil {
		t.Fatalf("Remove(%q): %v", name, err)
	}
}

// TestImplicitMkdir tests automatic parent directory creation.
func testImplicitMkdir(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if filesystem supports MkdirFS
	_, hasMkdirFS := fsys.(fs.MkdirFS)
	if !hasMkdirFS {
		t.Skip("MkdirFS not supported (required for implicit mkdir)")
	}

	const name = "auto_dir/nested/file.txt"
	testData := []byte("implicit mkdir test")

	// Create file in non-existent directory (should auto-create parent)
	if err := fs.WriteFile(ctx, fsys, name, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) with implicit mkdir: %v", name, err)
	}

	// Verify file can be read
	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	if !bytes.Equal(readData, testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", name, readData, testData)
	}

	// Verify parent directory exists
	info, err := fs.Stat(ctx, fsys, "auto_dir/nested")
	if err != nil {
		if !errors.Is(err, fs.ErrUnsupported) {
			t.Errorf(
				"Stat(%q) after implicit mkdir: %v", "auto_dir/nested", err,
			)
		}
	} else if !info.IsDir() {
		t.Errorf("Stat(%q): IsDir() = false, want true", "auto_dir/nested")
	}

	// Test with fs.Create as well
	const name2 = "auto_dir2/file2.txt"
	f, err := fs.Create(ctx, fsys, name2)
	if err != nil {
		t.Fatalf("Create(%q) with implicit mkdir: %v", name2, err)
	}

	_, writeErr := f.Write([]byte("created"))
	closeErr := f.Close()
	if writeErr != nil {
		t.Fatalf("Write after implicit mkdir: %v", writeErr)
	}
	if closeErr != nil {
		t.Fatalf("Close after implicit mkdir: %v", closeErr)
	}

	// Clean up
	if err := fs.RemoveAll(ctx, fsys, "auto_dir"); err != nil {
		t.Fatalf("RemoveAll(%q): %v", "auto_dir", err)
	}
	if err := fs.RemoveAll(ctx, fsys, "auto_dir2"); err != nil {
		t.Fatalf("RemoveAll(%q): %v", "auto_dir2", err)
	}
}

// TestImplicitMkdirWithMode tests WithDirMode context key.
func testImplicitMkdirWithMode(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	t.Helper()

	// Check if filesystem supports MkdirFS
	_, hasMkdirFS := fsys.(fs.MkdirFS)
	if !hasMkdirFS {
		t.Skip("MkdirFS not supported (required for implicit mkdir)")
	}

	const name = "custom_mode_dir/file.txt"
	testData := []byte("custom mode test")

	// Create file with custom directory mode
	ctxWithMode := fs.WithDirMode(ctx, 0700)
	err := fs.WriteFile(ctxWithMode, fsys, name, testData)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) with custom dir mode: %v", name, err)
	}

	// Verify file can be read
	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	if !bytes.Equal(readData, testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", name, readData, testData)
	}

	// If filesystem supports ChmodFS and StatFS, verify permissions
	_, hasChmod := fsys.(fs.ChmodFS)
	_, hasStat := fsys.(fs.StatFS)
	if hasChmod && hasStat {
		info, statErr := fs.Stat(ctx, fsys, "custom_mode_dir")
		if statErr != nil {
			if !errors.Is(statErr, fs.ErrUnsupported) {
				t.Errorf("Stat(%q): %v", "custom_mode_dir", statErr)
			}
		} else {
			// Check mode (accounting for directory bit and umask)
			mode := info.Mode()
			if !mode.IsDir() {
				t.Errorf(
					"Stat(%q): IsDir() = false, want true",
					"custom_mode_dir",
				)
			}
			// Verify permissions are 0700 (mode & 0777)
			perm := mode.Perm()
			if perm != 0700 {
				t.Logf(
					"Directory mode %04o != 0700 (may be umask)",
					perm,
				)
			}
		}
	}

	// Clean up
	if err := fs.RemoveAll(ctx, fsys, "custom_mode_dir"); err != nil {
		t.Fatalf("RemoveAll(%q): %v", "custom_mode_dir", err)
	}
}

// TestAppend tests fs.Append.
func testAppend(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	const name = "test_append.txt"

	// Create file with initial content
	initialData := []byte("initial")
	if err := fs.WriteFile(ctx, fsys, name, initialData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}

	// Append more data
	f, err := fs.Append(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Logf("Append not supported, skipping append test")
			// Clean up
			if removeErr := fs.Remove(ctx, fsys, name); removeErr != nil {
				t.Fatalf("Remove(%q): %v", name, removeErr)
			}
			return
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

	// Read back and verify both parts are there
	data, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	want := "initial appended"
	if string(data) != want {
		t.Errorf("ReadFile(%q) after Append = %q, want %q", name, data, want)
	}

	// Clean up
	if err := fs.Remove(ctx, fsys, name); err != nil {
		t.Fatalf("Remove(%q): %v", name, err)
	}
}
