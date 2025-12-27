package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleCreate() {
	fsys, ctx := osfs.NewTemp(), context.Background()
	defer fs.Close(fsys)

	f, err := fs.Create(ctx, fsys, "newfile.txt")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write([]byte("Creating a new file"))
	if err != nil {
		_ = f.Close()
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	data, err := fs.ReadFile(ctx, fsys, "newfile.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// Creating a new file
}
