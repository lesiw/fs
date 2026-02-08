package fstest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"lesiw.io/fs"
)

func testCreate(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("CreateAndRead", func(t *testing.T) {
		testCreateAndRead(ctx, t, fsys)
	})
	t.Run("CreateTruncates", func(t *testing.T) {
		testCreateTruncates(ctx, t, fsys)
	})
	t.Run("CreateBinaryData", func(t *testing.T) {
		testCreateBinaryData(ctx, t, fsys)
	})
	t.Run("WriteFileAndRead", func(t *testing.T) {
		testWriteFileAndRead(ctx, t, fsys)
	})
	t.Run("WriteFileOverwrite", func(t *testing.T) {
		testWriteFileOverwrite(ctx, t, fsys)
	})
	t.Run("WriteFileBinaryData", func(t *testing.T) {
		testWriteFileBinaryData(ctx, t, fsys)
	})
	t.Run("WriteFileCreatesParent", func(t *testing.T) {
		testWriteFileCreatesParent(ctx, t, fsys)
	})
	t.Run("CreateCreatesParent", func(t *testing.T) {
		testCreateCreatesParent(ctx, t, fsys)
	})
	t.Run("VirtualDirectoriesWithMode", func(t *testing.T) {
		testVirtualDirectoriesWithMode(ctx, t, fsys)
	})
}

func testCreateAndRead(ctx context.Context, t *testing.T, fsys fs.FS) {
	name := "test_create.txt"
	testData := []byte("hello world")

	f, err := fs.Create(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("Create(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	n, err := f.Write(testData)
	if err != nil {
		_ = f.Close()
		t.Fatalf("Write(%q): %v", testData, err)
	}
	if n != len(testData) {
		_ = f.Close()
		t.Fatalf("Write(%q) = %d bytes, want %d", testData, n, len(testData))
	}

	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}

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
}

func testCreateTruncates(ctx context.Context, t *testing.T, fsys fs.FS) {
	name := "test_create_truncate.txt"

	origData := []byte("original data")
	if err := fs.WriteFile(ctx, fsys, name, origData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	f, err := fs.Create(ctx, fsys, name)
	if err != nil {
		t.Fatalf("Create(%q) for truncate: %v", name, err)
	}
	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}

	data, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q) after truncate: %v", name, err)
	}
	if len(data) != 0 {
		t.Errorf("ReadFile(%q) after truncate = %d bytes, want 0",
			name, len(data))
	}
}

func testCreateBinaryData(ctx context.Context, t *testing.T, fsys fs.FS) {
	binaryData := make([]byte, 256)
	for i := range binaryData {
		binaryData[i] = byte(i)
	}

	name := "test_binary_create.bin"

	f, err := fs.Create(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("Create(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	n, err := f.Write(binaryData)
	if err != nil {
		_ = f.Close()
		t.Fatalf("Write(binary): %v", err)
	}
	if n != len(binaryData) {
		_ = f.Close()
		t.Fatalf("Write(binary) = %d bytes, want %d", n, len(binaryData))
	}

	if cerr := f.Close(); cerr != nil {
		t.Fatalf("Close(): %v", cerr)
	}

	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	if !bytes.Equal(readData, binaryData) {
		t.Errorf("Binary data corrupted: got %d bytes", len(readData))
		for i := 0; i < len(binaryData) && i < len(readData); i++ {
			if binaryData[i] != readData[i] {
				t.Errorf("First diff at byte %d: got 0x%02x, want 0x%02x",
					i, readData[i], binaryData[i])
				break
			}
		}
	}
}

func testWriteFileAndRead(ctx context.Context, t *testing.T, fsys fs.FS) {
	name := "test_write.txt"
	testData := []byte("test data for writefile")

	if err := fs.WriteFile(ctx, fsys, name, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	if !bytes.Equal(readData, testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", name, readData, testData)
	}
}

func testWriteFileOverwrite(ctx context.Context, t *testing.T, fsys fs.FS) {
	name := "test_write_overwrite.txt"
	initialData := []byte("initial data")

	if err := fs.WriteFile(ctx, fsys, name, initialData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	newData := []byte("new")
	writeErr := fs.WriteFile(ctx, fsys, name, newData)
	if writeErr != nil {
		t.Fatalf("WriteFile(%q) overwrite: %v", name, writeErr)
	}

	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q) after overwrite: %v", name, err)
	}

	if !bytes.Equal(readData, newData) {
		t.Errorf("ReadFile(%q) after overwrite = %q, want %q",
			name, readData, newData)
	}
}

func testWriteFileBinaryData(ctx context.Context, t *testing.T, fsys fs.FS) {
	binaryData := make([]byte, 256)
	for i := range binaryData {
		binaryData[i] = byte(i)
	}

	name := "test_binary_writefile.bin"

	if err := fs.WriteFile(ctx, fsys, name, binaryData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	if !bytes.Equal(readData, binaryData) {
		t.Errorf("Binary data corrupted: got %d bytes", len(readData))
		for i := 0; i < len(binaryData) && i < len(readData); i++ {
			if binaryData[i] != readData[i] {
				t.Errorf("First diff at byte %d: got 0x%02x, want 0x%02x",
					i, readData[i], binaryData[i])
				break
			}
		}
	}
}

func testWriteFileCreatesParent(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	_, hasMkdirFS := fsys.(fs.MkdirFS)
	if !hasMkdirFS {
		t.Skip("MkdirFS not supported (required for virtual directories)")
	}

	name := "auto_dir/nested/file.txt"
	testData := []byte("virtual directories test")

	if err := fs.WriteFile(ctx, fsys, name, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) with virtual directories: %v", name, err)
	}
	cleanup(ctx, t, fsys, "auto_dir")

	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	if !bytes.Equal(readData, testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", name, readData, testData)
	}

	info, err := fs.Stat(ctx, fsys, "auto_dir/nested")
	if err != nil {
		if !errors.Is(err, fs.ErrUnsupported) {
			t.Errorf("Stat(%q) after virtual directories: %v",
				"auto_dir/nested", err)
		}
	} else if !info.IsDir() {
		t.Errorf("Stat(%q): IsDir() = false, want true", "auto_dir/nested")
	}
}

func testCreateCreatesParent(ctx context.Context, t *testing.T, fsys fs.FS) {
	_, hasMkdirFS := fsys.(fs.MkdirFS)
	if !hasMkdirFS {
		t.Skip("MkdirFS not supported (required for virtual directories)")
	}

	name := "auto_dir2/file2.txt"
	f, err := fs.Create(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("Create(%q) with virtual directories: %v", name, err)
	}
	cleanup(ctx, t, fsys, "auto_dir2")

	_, writeErr := f.Write([]byte("created"))
	closeErr := f.Close()
	if writeErr != nil {
		t.Fatalf("Write after virtual directories: %v", writeErr)
	}
	if closeErr != nil {
		t.Fatalf("Close after virtual directories: %v", closeErr)
	}
}

func testVirtualDirectoriesWithMode(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	_, hasMkdirFS := fsys.(fs.MkdirFS)
	if !hasMkdirFS {
		t.Skip("MkdirFS not supported (required for virtual directories)")
	}

	name := "custom_mode_dir/file.txt"
	testData := []byte("custom mode test")

	ctxWithMode := fs.WithDirMode(ctx, 0700)
	err := fs.WriteFile(ctxWithMode, fsys, name, testData)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q) with custom dir mode: %v", name, err)
	}
	cleanup(ctx, t, fsys, "custom_mode_dir")

	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	if !bytes.Equal(readData, testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", name, readData, testData)
	}

	_, hasChmod := fsys.(fs.ChmodFS)
	_, hasStat := fsys.(fs.StatFS)
	if hasChmod && hasStat {
		info, statErr := fs.Stat(ctx, fsys, "custom_mode_dir")
		if statErr != nil {
			if !errors.Is(statErr, fs.ErrUnsupported) {
				t.Errorf("Stat(%q): %v", "custom_mode_dir", statErr)
			}
		} else {
			mode := info.Mode()
			if !mode.IsDir() {
				t.Errorf("Stat(%q): IsDir() = false, want true",
					"custom_mode_dir")
			}
			perm := mode.Perm()
			if perm != 0700 {
				t.Logf("Directory mode %04o != 0700 (may be umask)", perm)
			}
		}
	}
}
