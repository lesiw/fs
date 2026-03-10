package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func testFindUp(
	ctx context.Context, t *testing.T, fsys fs.FS, files []File,
) {
	var deepDir, rootFile string
	for _, f := range files {
		if d := path.Dir(f.Path); deepDir == "" && d != "" {
			deepDir = d
		}
		if rootFile == "" && path.Dir(f.Path) == "" {
			rootFile = f.Path
		}
	}
	if deepDir == "" || rootFile == "" {
		t.Skip("need a nested dir and a root-level file")
	}

	dir, err := fs.Abs(fs.WithWorkDir(ctx, deepDir), fsys, ".")
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Abs not supported")
		}
		t.Fatalf("Abs: %v", err)
	}
	ctx = fs.WithWorkDir(ctx, dir)
	for {
		if _, err := fs.Stat(ctx, fsys, path.Join(dir, rootFile)); err == nil {
			return
		}
		if p := path.Dir(dir); p != dir {
			dir = p
		} else {
			t.Fatal("walked to root without finding " + rootFile)
		}
	}
}

func testFindUpDotDot(
	ctx context.Context, t *testing.T, fsys fs.FS, files []File,
) {
	var deepDir, rootFile string
	for _, f := range files {
		if d := path.Dir(f.Path); deepDir == "" && d != "" {
			deepDir = d
		}
		if rootFile == "" && path.Dir(f.Path) == "" {
			rootFile = f.Path
		}
	}
	if deepDir == "" || rootFile == "" {
		t.Skip("need a nested dir and a root-level file")
	}

	dir, err := fs.Abs(fs.WithWorkDir(ctx, deepDir), fsys, ".")
	if err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("Abs not supported")
		}
		t.Fatalf("Abs: %v", err)
	}
	ctx = fs.WithWorkDir(ctx, dir)
	for {
		if _, err := fs.Stat(ctx, fsys, rootFile); err == nil {
			return
		}
		ctx = fs.WithWorkDir(ctx, "..")
		if p := fs.WorkDir(ctx); p != dir {
			dir = p
		} else {
			t.Fatal("walked to root without finding " + rootFile)
		}
	}
}
