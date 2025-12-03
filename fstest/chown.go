package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testChown(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("Chown", func(t *testing.T) {
		fileName := "test_chown_file.txt"
		testData := []byte("chown test")
		if err := fs.WriteFile(ctx, fsys, fileName, testData); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("WriteFile(%q): %v", fileName, err)
		}
		cleanup(ctx, t, fsys, fileName)

		_, statErr := fs.Stat(ctx, fsys, fileName)
		if statErr != nil {
			t.Fatalf("Stat(%q): %v", fileName, statErr)
		}

		// Most systems don't allow non-root users to change ownership to
		// different users, so we use -1, -1 which means "don't change"
		err := fs.Chown(ctx, fsys, fileName, -1, -1)

		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Chown not supported")
			}
			t.Fatalf("Chown(%q, -1, -1): %v", fileName, err)
		}

		data, readErr := fs.ReadFile(ctx, fsys, fileName)
		if readErr != nil {
			t.Fatalf("ReadFile(%q) after chown: %v", fileName, readErr)
		}

		if string(data) != string(testData) {
			t.Errorf(
				"ReadFile(%q) after chown = %q, want %q",
				fileName, data, testData,
			)
		}
	})
}
