package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleTruncate() {
	fsys, ctx := osfs.NewTemp(), context.Background()
	defer fs.Close(fsys)

	err := fs.WriteFile(ctx, fsys, "shrink.txt", []byte("Hello, World!"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.Truncate(ctx, fsys, "shrink.txt", 5)
	if err != nil {
		log.Fatal(err)
	}
	data, err := fs.ReadFile(ctx, fsys, "shrink.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// Hello
}
