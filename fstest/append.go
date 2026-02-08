package fstest

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testAppend(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("AppendAndRead", func(t *testing.T) {
		testAppendAndRead(ctx, t, fsys)
	})
	t.Run("AppendBinaryData", func(t *testing.T) {
		testAppendBinaryData(ctx, t, fsys)
	})
	t.Run("AppendCreatesFile", func(t *testing.T) {
		testAppendCreatesFile(ctx, t, fsys)
	})
	t.Run("AppendCreatesParent", func(t *testing.T) {
		testAppendCreatesParent(ctx, t, fsys)
	})
}

func testAppendAndRead(ctx context.Context, t *testing.T, fsys fs.FS) {
	name := "test_append.txt"

	initialData := []byte("initial")
	if err := fs.WriteFile(ctx, fsys, name, initialData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	f, err := fs.Append(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Append not supported")
		}
		t.Fatalf("Append(%q): %v", name, err)
	}

	if _, writeErr := f.Write([]byte(" appended")); writeErr != nil {
		_ = f.Close()
		t.Fatalf("Write(%q): %v", " appended", writeErr)
	}

	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}

	data, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	want := "initial appended"
	if string(data) != want {
		t.Errorf("ReadFile(%q) after Append = %q, want %q", name, data, want)
	}
}

func testAppendBinaryData(ctx context.Context, t *testing.T, fsys fs.FS) {
	firstHalf := make([]byte, 128)
	for i := range firstHalf {
		firstHalf[i] = byte(i)
	}

	secondHalf := make([]byte, 128)
	for i := range secondHalf {
		secondHalf[i] = byte(128 + i)
	}

	name := "test_append_binary.bin"

	if err := fs.WriteFile(ctx, fsys, name, firstHalf); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	f, err := fs.Append(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Append not supported")
		}
		t.Fatalf("Append(%q): %v", name, err)
	}

	n, err := f.Write(secondHalf)
	if err != nil {
		_ = f.Close()
		t.Fatalf("Write(binary): %v", err)
	}
	if n != len(secondHalf) {
		_ = f.Close()
		t.Fatalf("Write(binary) = %d bytes, want %d", n, len(secondHalf))
	}

	if cerr := f.Close(); cerr != nil {
		t.Fatalf("Close(): %v", cerr)
	}

	readData, err := fs.ReadFile(ctx, fsys, name)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", name, err)
	}

	expected := append(firstHalf, secondHalf...)
	if !bytes.Equal(readData, expected) {
		t.Errorf("Binary data corrupted: got %d bytes, want %d",
			len(readData), len(expected))
		for i := 0; i < len(expected) && i < len(readData); i++ {
			if expected[i] != readData[i] {
				t.Errorf("First diff at byte %d: got 0x%02x, want 0x%02x",
					i, readData[i], expected[i])
				break
			}
		}
	}
}

func testAppendCreatesFile(ctx context.Context, t *testing.T, fsys fs.FS) {
	name := "test_append_creates.txt"

	f, err := fs.Append(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Append not supported")
		}
		t.Fatalf("Append(%q) on non-existent file: %v", name, err)
	}
	cleanup(ctx, t, fsys, name)

	testData := []byte("created by append")
	if _, writeErr := f.Write(testData); writeErr != nil {
		_ = f.Close()
		t.Fatalf("Write(%q): %v", testData, writeErr)
	}

	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}

	data, readErr := fs.ReadFile(ctx, fsys, name)
	if readErr != nil {
		t.Fatalf("ReadFile(%q): %v", name, readErr)
	}

	if !bytes.Equal(data, testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", name, data, testData)
	}
}

func testAppendCreatesParent(ctx context.Context, t *testing.T, fsys fs.FS) {
	_, hasMkdirFS := fsys.(fs.MkdirFS)
	if !hasMkdirFS {
		t.Skip("MkdirFS not supported (required for virtual directories)")
	}

	name := "append_auto_dir/nested/file.txt"
	cleanup(ctx, t, fsys, "append_auto_dir")

	f, err := fs.Append(ctx, fsys, name)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Append not supported")
		}
		t.Fatalf("Append(%q) with virtual directories: %v", name, err)
	}

	testData := []byte("auto-created parents")
	if _, writeErr := f.Write(testData); writeErr != nil {
		_ = f.Close()
		t.Fatalf("Write(%q): %v", testData, writeErr)
	}

	if closeErr := f.Close(); closeErr != nil {
		t.Fatalf("Close(): %v", closeErr)
	}

	data, readErr := fs.ReadFile(ctx, fsys, name)
	if readErr != nil {
		t.Fatalf("ReadFile(%q): %v", name, readErr)
	}

	if !bytes.Equal(data, testData) {
		t.Errorf("ReadFile(%q) = %q, want %q", name, data, testData)
	}

	info, statErr := fs.Stat(ctx, fsys, "append_auto_dir/nested")
	if statErr != nil {
		if !errors.Is(statErr, fs.ErrUnsupported) {
			t.Errorf("Stat(%q) after virtual directories: %v",
				"append_auto_dir/nested", statErr)
		}
	} else if !info.IsDir() {
		t.Errorf("Stat(%q): IsDir() = false, want true",
			"append_auto_dir/nested")
	}
}
