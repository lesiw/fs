package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
)

func testSymlink(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("SymlinkFile", func(t *testing.T) {
		testSymlinkFile(ctx, t, fsys)
	})

	t.Run("SymlinkDir", func(t *testing.T) {
		testSymlinkDir(ctx, t, fsys)
	})

	t.Run("ReadLink", func(t *testing.T) {
		testReadlink(ctx, t, fsys)
	})
}

func testSymlinkFile(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	targetFile := "test_symlink_target_file.txt"
	testData := []byte("symlink target content")
	if err := fs.WriteFile(ctx, fsys, targetFile, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", targetFile, err)
	}
	cleanup(ctx, t, fsys, targetFile)

	linkName := "test_symlink_link_file.txt"
	err := fs.Symlink(ctx, fsys, targetFile, linkName)

	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Symlink not supported")
		}
		t.Fatalf("Symlink(%q, %q): %v", targetFile, linkName, err)
	}
	cleanup(ctx, t, fsys, linkName)

	data, readErr := fs.ReadFile(ctx, fsys, linkName)
	if readErr != nil {
		t.Fatalf("ReadFile(%q) through symlink: %v", linkName, readErr)
	}

	if string(data) != string(testData) {
		t.Errorf(
			"ReadFile(%q) through symlink = %q, want %q",
			linkName, data, testData,
		)
	}
}

func testSymlinkDir(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	targetDir := "test_symlink_target_dir"
	mkdirErr := fs.Mkdir(ctx, fsys, targetDir)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported")
	}
	if mkdirErr != nil {
		t.Fatalf("Mkdir(%q): %v", targetDir, mkdirErr)
	}
	cleanup(ctx, t, fsys, targetDir)

	dirLink := "test_symlink_link_dir"
	err := fs.Symlink(ctx, fsys, targetDir, dirLink)
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Symlink not supported")
		}
		t.Fatalf("Symlink(%q, %q): %v", targetDir, dirLink, err)
	}
	cleanup(ctx, t, fsys, dirLink)

	info, statErr := fs.Stat(ctx, fsys, dirLink)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", dirLink, statErr)
	}

	if !info.IsDir() {
		t.Errorf(
			"Stat(%q) through symlink: IsDir() = false, want true",
			dirLink,
		)
	}
}

func testReadlink(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	targetFile := "test_readlink_target.txt"
	testData := []byte("target")
	if err := fs.WriteFile(ctx, fsys, targetFile, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", targetFile, err)
	}
	cleanup(ctx, t, fsys, targetFile)

	linkName := "test_readlink_link.txt"
	symlinkErr := fs.Symlink(ctx, fsys, targetFile, linkName)
	if symlinkErr != nil {
		if errors.Is(symlinkErr, fs.ErrUnsupported) {
			t.Skip("Symlink not supported (required for Readlink test)")
		}
		t.Fatalf("Symlink(%q, %q): %v", targetFile, linkName, symlinkErr)
	}
	cleanup(ctx, t, fsys, linkName)

	target, err := fs.ReadLink(ctx, fsys, linkName)

	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Readlink not supported")
		}
		t.Fatalf("Readlink(%q): %v", linkName, err)
	}

	if target != targetFile {
		t.Errorf(
			"Readlink(%q) = %q, want %q", linkName, target, targetFile,
		)
	}
}
