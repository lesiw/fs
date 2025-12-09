package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleReadFile() {
	fsys, ctx := osfs.TempFS(), context.Background()
	defer fs.Close(fsys)

	content := []byte("Hello, World!")
	err := fs.WriteFile(ctx, fsys, "greeting.txt", content)
	if err != nil {
		log.Fatal(err)
	}
	data, err := fs.ReadFile(ctx, fsys, "greeting.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// Hello, World!
}
