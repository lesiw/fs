package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testTruncate(ctx context.Context, t *testing.T, fsys fs.FS) {
	_, hasTruncate := fsys.(fs.TruncateFS)
	_, hasRemove := fsys.(fs.RemoveFS)
	_, hasCreate := fsys.(fs.CreateFS)

	if !hasTruncate && !hasRemove {
		t.Skip(
			"Truncate not supported " +
				"(requires TruncateFS or RemoveFS+CreateFS)",
		)
	}
	if !hasTruncate && !hasCreate {
		t.Skip(
			"Truncate not supported " +
				"(requires TruncateFS or RemoveFS+CreateFS)",
		)
	}
	t.Run("TruncateShrink", func(t *testing.T) {
		testTruncateShrink(ctx, t, fsys)
	})
	t.Run("TruncateExpand", func(t *testing.T) {
		testTruncateExpand(ctx, t, fsys)
	})
	t.Run("TruncateBinaryData", func(t *testing.T) {
		testTruncateBinaryData(ctx, t, fsys)
	})
}

func testTruncateShrink(ctx context.Context, t *testing.T, fsys fs.FS) {
	fileName := "test_truncate_shrink.txt"
	testData := []byte("hello world this is a test")
	if err := fs.WriteFile(ctx, fsys, fileName, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", fileName, err)
	}
	cleanup(ctx, t, fsys, fileName)

	newSize := int64(5)
	err := fs.Truncate(ctx, fsys, fileName, newSize)
	if err != nil {
		t.Fatalf("Truncate(%q, %d): %v", fileName, newSize, err)
	}

	info, statErr := fs.Stat(ctx, fsys, fileName)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrUnsupported) {
			t.Log("Stat not supported, skipping size verification")
		} else {
			t.Fatalf("Stat(%q): %v", fileName, statErr)
		}
	} else if info.Size() != newSize {
		t.Errorf(
			"Truncate(%q, %d): Size() = %d, want %d",
			fileName, newSize, info.Size(), newSize,
		)
	}

	data, readErr := fs.ReadFile(ctx, fsys, fileName)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) after truncate: %v", fileName, readErr)
	}

	expected := testData[:newSize]
	if string(data) != string(expected) {
		t.Errorf(
			"ReadFile(%q) after truncate = %q, want %q",
			fileName, data, expected,
		)
	}
}

func testTruncateExpand(ctx context.Context, t *testing.T, fsys fs.FS) {
	fileName := "test_truncate_expand.txt"
	testData := []byte("hello")
	if err := fs.WriteFile(ctx, fsys, fileName, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", fileName, err)
	}
	cleanup(ctx, t, fsys, fileName)

	largerSize := int64(10)
	if err := fs.Truncate(ctx, fsys, fileName, largerSize); err != nil {
		t.Fatalf("Truncate(%q, %d): %v", fileName, largerSize, err)
	}

	info, statErr := fs.Stat(ctx, fsys, fileName)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrUnsupported) {
			t.Log("Stat not supported, skipping size verification")
		} else {
			t.Fatalf("Stat(%q): %v", fileName, statErr)
		}
	} else if info.Size() != largerSize {
		t.Errorf(
			"Truncate(%q, %d): Size() = %d, want %d",
			fileName, largerSize, info.Size(), largerSize,
		)
	}
}

func testTruncateBinaryData(ctx context.Context, t *testing.T, fsys fs.FS) {
	binaryData := make([]byte, 256)
	for i := range binaryData {
		binaryData[i] = byte(i)
	}

	fileName := "test_truncate_binary.bin"
	if err := fs.WriteFile(ctx, fsys, fileName, binaryData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", fileName, err)
	}
	cleanup(ctx, t, fsys, fileName)

	newSize := int64(128)
	if err := fs.Truncate(ctx, fsys, fileName, newSize); err != nil {
		t.Fatalf("Truncate(%q, %d): %v", fileName, newSize, err)
	}

	data, err := fs.ReadFile(ctx, fsys, fileName)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", fileName, err)
	}

	expected := binaryData[:newSize]
	if len(data) != len(expected) {
		t.Fatalf("ReadFile(%q) = %d bytes, want %d",
			fileName, len(data), len(expected))
	}

	for i := 0; i < len(expected); i++ {
		if data[i] != expected[i] {
			t.Errorf(
				"Binary data corrupted at byte %d: got 0x%02x, want 0x%02x",
				i, data[i], expected[i],
			)
			break
		}
	}
}
