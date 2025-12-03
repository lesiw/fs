package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testTruncate(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
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

	t.Run("Truncate", func(t *testing.T) {
		fileName := "test_truncate_shrink.txt"
		testData := []byte("hello world this is a test")
		if err := fs.WriteFile(ctx, fsys, fileName, testData); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("WriteFile(%q): %v", fileName, err)
		}
		cleanup(ctx, t, fsys, fileName)

		// Truncate to smaller size
		newSize := int64(5)
		err := fs.Truncate(ctx, fsys, fileName, newSize)
		if err != nil {
			t.Fatalf("Truncate(%q, %d): %v", fileName, newSize, err)
		}

		// Verify new size (skip size check if Stat not supported)
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

		// Verify content
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
	})

	t.Run("TruncateExpand", func(t *testing.T) {
		fileName := "test_truncate_expand.txt"
		testData := []byte("hello")
		if err := fs.WriteFile(ctx, fsys, fileName, testData); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("WriteFile(%q): %v", fileName, err)
		}
		cleanup(ctx, t, fsys, fileName)

		// Truncate to larger size (should extend with zeros)
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
	})
}
