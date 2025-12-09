package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleTemp_dir() {
	fsys, ctx := osfs.TempFS(), context.Background()
	defer fs.Close(fsys)

	// Create temp directory (trailing slash indicates directory)
	w, err := fs.Temp(ctx, fsys, "myapp/")
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	// Get the directory path and defer cleanup
	dir := w.Path()
	defer fs.RemoveAll(ctx, fsys, dir)

	// Create a file in the temp directory
	err = fs.WriteFile(ctx, fsys, dir+"/data.txt", []byte("temporary data"))
	if err != nil {
		log.Fatal(err)
	}

	// Read it back
	data, err := fs.ReadFile(ctx, fsys, dir+"/data.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// temporary data
}

func ExampleTemp_file() {
	fsys, ctx := osfs.TempFS(), context.Background()
	defer fs.Close(fsys)

	// Create temp file (no trailing slash)
	w, err := fs.Temp(ctx, fsys, "myapp")
	if err != nil {
		log.Fatal(err)
	}

	// Get the file path and defer cleanup
	path := w.Path()
	defer fs.Remove(ctx, fsys, path)

	// Write to the temp file
	_, err = w.Write([]byte("temporary data"))
	if err != nil {
		_ = w.Close()
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	// Read it back
	data, err := fs.ReadFile(ctx, fsys, path)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// temporary data
}
