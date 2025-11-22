package fstest

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"lesiw.io/fs"
)

func testAbs(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	t.Run("AlreadyAbsolute", func(t *testing.T) {
		abs, err := fs.Abs(ctx, fsys, "/already/absolute")
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Abs not supported")
			}
			t.Fatalf("Abs(/already/absolute) err: %v", err)
		}
		// Normalize to forward slashes for cross-platform comparison.
		// Absolute paths are platform-specific but should end with the path.
		got, want := filepath.ToSlash(abs), "/already/absolute"
		if !strings.HasSuffix(got, want) {
			t.Errorf("Abs(/already/absolute) = %q, want suffix %q", abs, want)
		}
	})

	t.Run("RelativePath", func(t *testing.T) {
		input := "relative/path"
		got, err := fs.Abs(ctx, fsys, input)
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Abs not supported")
			}
			t.Fatalf("Abs(relative/path) err: %v", err)
		}
		if got == input {
			t.Errorf("Abs(relative/path) = %q, want different from input", got)
		}
	})

	t.Run("WithAbsoluteWorkDir", func(t *testing.T) {
		wctx := fs.WithWorkDir(ctx, "/absolute/workdir")
		abs, err := fs.Abs(wctx, fsys, "file.txt")
		if err != nil {
			// This should work via fallback even without AbsFS.
			t.Fatalf("Abs(file.txt) err: %v", err)
		}
		got, want := filepath.ToSlash(abs), "workdir"
		if !strings.Contains(got, want) {
			t.Errorf(
				"Abs(file.txt) with WorkDir=/absolute/workdir = %q, "+
					"want to contain %q",
				abs, want,
			)
		}
	})

	t.Run("WithRelativeWorkDir", func(t *testing.T) {
		wctx := fs.WithWorkDir(ctx, "relative/workdir")
		abs, err := fs.Abs(wctx, fsys, "file.txt")
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Abs not supported")
			}
			t.Fatalf("Abs(file.txt) err: %v", err)
		}
		got, want := filepath.ToSlash(abs), "workdir"
		if !strings.Contains(got, want) {
			t.Errorf(
				"Abs(file.txt) with WorkDir=relative/workdir = %q, "+
					"want to contain %q",
				abs, want,
			)
		}
	})

	t.Run("WorkDirAffectsResult", func(t *testing.T) {
		noWork, err := fs.Abs(ctx, fsys, "file.txt")
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Abs not supported")
			}
			t.Fatalf("Abs(file.txt) err: %v", err)
		}
		wctx := fs.WithWorkDir(ctx, "subdir")
		withWork, err := fs.Abs(wctx, fsys, "file.txt")
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("Abs not supported")
			}
			t.Fatalf("Abs(file.txt) err: %v", err)
		}
		if withWork == noWork {
			t.Errorf(
				"Abs(file.txt) with WorkDir=subdir = %q, "+
					"without WorkDir = %q, want different values",
				withWork, noWork,
			)
		}
		got, want := filepath.ToSlash(withWork), "subdir"
		if !strings.Contains(got, want) {
			t.Errorf(
				"Abs(file.txt) with WorkDir=subdir = %q, want to contain %q",
				withWork, want,
			)
		}
	})
}
