package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleMkdir() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	err = fs.Mkdir(ctx, fsys, "mydir")
	if err != nil {
		log.Fatal(err)
	}
	info, err := fs.Stat(ctx, fsys, "mydir")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created: %s (isDir: %v)\n", info.Name(), info.IsDir())
	// Output:
	// Created: mydir (isDir: true)
}

func ExampleMkdirAll() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	err = fs.MkdirAll(ctx, fsys, "path/to/nested/dir")
	if err != nil {
		log.Fatal(err)
	}
	info, err := fs.Stat(ctx, fsys, "path/to/nested/dir")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created: %v\n", info.IsDir())
	// Output:
	// Created: true
}
