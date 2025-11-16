package fs_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleRemove() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	err = fs.WriteFile(ctx, fsys, "delete-me.txt", []byte("temporary"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.Remove(ctx, fsys, "delete-me.txt")
	if err != nil {
		log.Fatal(err)
	}
	_, err = fs.Stat(ctx, fsys, "delete-me.txt")
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Println("File successfully removed")
	}
	// Output:
	// File successfully removed
}

func ExampleRemoveAll() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	err = fs.MkdirAll(ctx, fsys, "tree/branch/leaf")
	if err != nil {
		log.Fatal(err)
	}
	err = fs.WriteFile(ctx, fsys, "tree/file.txt", []byte("data"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.RemoveAll(ctx, fsys, "tree")
	if err != nil {
		log.Fatal(err)
	}
	_, err = fs.Stat(ctx, fsys, "tree")
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Println("Directory tree successfully removed")
	}
	// Output:
	// Directory tree successfully removed
}
