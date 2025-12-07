package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleWalk_recursive() {
	fsys, ctx := osfs.TempFS(context.Background())
	defer fs.Close(fsys)

	err := fs.MkdirAll(ctx, fsys, "walk/sub")
	if err != nil {
		log.Fatal(err)
	}
	err = fs.WriteFile(ctx, fsys, "walk/file1.txt", []byte("one"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.WriteFile(ctx, fsys, "walk/sub/file2.txt", []byte("two"))
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for entry, err := range fs.Walk(ctx, fsys, "walk", -1) {
		if err != nil {
			log.Fatal(err)
		}
		if !entry.IsDir() {
			count++
		}
	}
	fmt.Printf("Found %d files\n", count)
	// Output:
	// Found 2 files
}
