package fstest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func testTemp(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("TempFile", func(t *testing.T) {
		testTempFile(ctx, t, fsys)
	})

	t.Run("TempDir", func(t *testing.T) {
		testTempDir(ctx, t, fsys)
	})
}

func testTempFile(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	t.Run("CreateAndWrite", func(t *testing.T) {
		prefix := "test_tempfile"
		w, err := fs.Temp(ctx, fsys, prefix)
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Temp not supported")
			}
			t.Fatalf("Temp(%q) err: %v", prefix, err)
		}
		cleanup(ctx, t, fsys, w.Path())

		tempFile := w.Path()
		if !strings.Contains(tempFile, prefix) {
			t.Errorf(
				"Temp(%q) path = %q, want to contain %q",
				prefix, tempFile, prefix,
			)
		}

		testData := []byte("temporary file content")
		if _, err := w.Write(testData); err != nil {
			t.Fatalf("Write err: %v", err)
		}

		if err := w.Close(); err != nil {
			t.Fatalf("Close err: %v", err)
		}
	})

	t.Run("UniqueNames", func(t *testing.T) {
		prefix := "test_unique"
		w1, err := fs.Temp(ctx, fsys, prefix)
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Temp not supported")
			}
			t.Fatalf("Temp(%q) err: %v", prefix, err)
		}
		cleanup(ctx, t, fsys, w1.Path())

		w2, err := fs.Temp(ctx, fsys, prefix)
		if err != nil {
			t.Fatalf("Temp(%q) second call err: %v", prefix, err)
		}
		cleanup(ctx, t, fsys, w2.Path())

		if err := w1.Close(); err != nil {
			t.Fatalf("Close w1 err: %v", err)
		}
		if err := w2.Close(); err != nil {
			t.Fatalf("Close w2 err: %v", err)
		}

		if pathsEqual([]string{w1.Path()}, []string{w2.Path()}) {
			t.Errorf(
				"Temp(%q) created duplicate names: %q",
				prefix, w1.Path(),
			)
		}
	})
}

func testTempDir(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("CreateAndUse", func(t *testing.T) {
		prefix := "test_tempdir/"
		w, err := fs.Temp(ctx, fsys, prefix)
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Temp not supported")
			}
			t.Fatalf("Temp(%q) err: %v", prefix, err)
		}
		cleanup(ctx, t, fsys, w.Path())

		tempDir := w.Path()
		prefixWithoutSlash := strings.TrimSuffix(prefix, "/")
		if !strings.Contains(tempDir, prefixWithoutSlash) {
			t.Errorf(
				"Temp(%q) path = %q, want to contain %q",
				prefix, tempDir, prefixWithoutSlash,
			)
		}

		if closeErr := w.Close(); closeErr != nil {
			t.Fatalf("Close err: %v", closeErr)
		}

		testFile := tempDir + "/test.txt"
		testData := []byte("temp file content")
		writeErr := fs.WriteFile(ctx, fsys, testFile, testData)
		if writeErr != nil {
			if errors.Is(writeErr, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("WriteFile(%q) err: %v", testFile, writeErr)
		}

		info, statErr := fs.Stat(ctx, fsys, tempDir)
		if statErr != nil {
			t.Fatalf("Stat(%q) err: %v", tempDir, statErr)
		}
		if !info.IsDir() {
			t.Errorf("Stat(%q) IsDir = false, want true", tempDir)
		}

		data, readErr := fs.ReadFile(ctx, fsys, testFile)
		if readErr != nil {
			t.Fatalf("ReadFile(%q) err: %v", testFile, readErr)
		}
		if string(data) != string(testData) {
			t.Errorf("ReadFile = %q, want %q", data, testData)
		}
	})

	t.Run("UniqueNames", func(t *testing.T) {
		prefix := "test_unique/"
		w1, err := fs.Temp(ctx, fsys, prefix)
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Temp not supported")
			}
			t.Fatalf("Temp(%q) err: %v", prefix, err)
		}
		cleanup(ctx, t, fsys, w1.Path())

		w2, err := fs.Temp(ctx, fsys, prefix)
		if err != nil {
			t.Fatalf("Temp(%q) second call err: %v", prefix, err)
		}
		cleanup(ctx, t, fsys, w2.Path())

		if pathsEqual([]string{w1.Path()}, []string{w2.Path()}) {
			t.Errorf(
				"Temp(%q) created duplicate names: %q",
				prefix, w1.Path(),
			)
		}
	})

	t.Run("PathSeparators", func(t *testing.T) {
		prefix := "test_separators/"
		w, err := fs.Temp(ctx, fsys, prefix)
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Temp not supported")
			}
			t.Fatalf("Temp(%q) err: %v", prefix, err)
		}
		w.Close()
		cleanup(ctx, t, fsys, w.Path())

		name := path.Join(w.Path(), "foo/bar.txt")

		hasForward := strings.ContainsRune(name, '/')
		hasBackward := strings.ContainsRune(name, '\\')
		if hasForward && hasBackward {
			t.Errorf("path.Join(%q, %q) returned %q "+
				"with mixed separators", w.Path(), "foo/bar.txt", name)
		}
	})
}
