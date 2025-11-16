//go:build unix

package fs_test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func TestContextModesIndependent(t *testing.T) {
	ctx := t.Context()

	// Set both modes
	ctx = fs.WithDirMode(ctx, 0700)
	ctx = fs.WithFileMode(ctx, 0600)

	// Verify both are preserved independently
	dirMode := fs.DirMode(ctx)
	fileMode := fs.FileMode(ctx)

	if dirMode != 0700 {
		t.Errorf("DirMode(ctx) = %04o, want 0700", dirMode)
	}

	if fileMode != 0600 {
		t.Errorf("FileMode(ctx) = %04o, want 0600", fileMode)
	}
}

func ExampleWithFileMode() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	ctx = fs.WithFileMode(ctx, 0600)
	err = fs.WriteFile(ctx, fsys, "private.txt", []byte("secret"))
	if err != nil {
		log.Fatal(err)
	}
	info, err := fs.Stat(ctx, fsys, "private.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Mode: %04o\n", info.Mode().Perm())
	// Output:
	// Mode: 0600
}

func ExampleWithDirMode() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	ctx = fs.WithDirMode(ctx, 0700)
	err = fs.MkdirAll(ctx, fsys, "private/data")
	if err != nil {
		log.Fatal(err)
	}
	info, err := fs.Stat(ctx, fsys, "private")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Mode: %04o\n", info.Mode().Perm())
	// Output:
	// Mode: 0700
}

func ExampleFileMode() {
	ctx := context.Background()
	ctx = fs.WithFileMode(ctx, 0600)
	mode := fs.FileMode(ctx)
	fmt.Printf("Mode: %04o\n", mode)
	// Output:
	// Mode: 0600
}

func ExampleDirMode() {
	ctx := context.Background()
	ctx = fs.WithDirMode(ctx, 0700)
	mode := fs.DirMode(ctx)
	fmt.Printf("Mode: %04o\n", mode)
	// Output:
	// Mode: 0700
}
