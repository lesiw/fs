package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleRename() {
	fsys, ctx := osfs.TempFS(), context.Background()
	defer fs.Close(fsys)

	err := fs.WriteFile(ctx, fsys, "old-name.txt", []byte("content"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.Rename(ctx, fsys, "old-name.txt", "new-name.txt")
	if err != nil {
		log.Fatal(err)
	}
	data, err := fs.ReadFile(ctx, fsys, "new-name.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Content: %s\n", data)
	// Output:
	// Content: content
}
