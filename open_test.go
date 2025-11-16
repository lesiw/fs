package fs_test

import (
	"context"
	"fmt"
	"io"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleOpen() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	err = fs.WriteFile(ctx, fsys, "data.txt", []byte("example content"))
	if err != nil {
		log.Fatal(err)
	}
	f, err := fs.Open(ctx, fsys, "data.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// example content
}
