package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleSymlink() {
	fsys, ctx := osfs.TempFS(context.Background())
	defer fs.Close(fsys)

	err := fs.WriteFile(ctx, fsys, "original.txt", []byte("content"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.Symlink(ctx, fsys, "original.txt", "link.txt")
	if err != nil {
		log.Fatal(err)
	}
	data, err := fs.ReadFile(ctx, fsys, "link.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// content
}

func ExampleReadLink() {
	fsys, ctx := osfs.TempFS(context.Background())
	defer fs.Close(fsys)

	err := fs.Symlink(ctx, fsys, "target.txt", "link.txt")
	if err != nil {
		log.Fatal(err)
	}
	target, err := fs.ReadLink(ctx, fsys, "link.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(target)
	// Output:
	// target.txt
}
