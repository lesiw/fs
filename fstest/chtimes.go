package fstest

import (
	"context"
	"errors"
	"testing"
	"time"

	"lesiw.io/fs"
)

func testChtimes(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("Chtimes", func(t *testing.T) {
		fileName := "test_chtimes_file.txt"
		testData := []byte("chtimes test")
		if err := fs.WriteFile(ctx, fsys, fileName, testData); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("WriteFile(%q): %v", fileName, err)
		}
		cleanup(ctx, t, fsys, fileName)

		atime := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
		mtime := time.Date(2021, 6, 15, 18, 30, 0, 0, time.UTC)

		err := fs.Chtimes(ctx, fsys, fileName, atime, mtime)

		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Chtimes not supported")
			}
			t.Fatalf("Chtimes(%q): %v", fileName, err)
		}

		info, statErr := fs.Stat(ctx, fsys, fileName)
		if statErr != nil {
			if errors.Is(statErr, fs.ErrUnsupported) {
				t.Skip("Stat not supported")
			}
			t.Fatalf("Stat(%q): %v", fileName, statErr)
		}

		// Allow 1 second tolerance for filesystem time granularity
		gotMtime := info.ModTime()
		if gotMtime.Sub(mtime).Abs() > time.Second {
			t.Errorf(
				"Chtimes(%q): ModTime() = %v, want %v",
				fileName, gotMtime, mtime,
			)
		}

		data, readErr := fs.ReadFile(ctx, fsys, fileName)
		if readErr != nil {
			t.Fatalf("ReadFile(%q) after chtimes: %v", fileName, readErr)
		}

		if string(data) != string(testData) {
			t.Errorf(
				"ReadFile(%q) after chtimes = %q, want %q",
				fileName, data, testData,
			)
		}
	})
}
