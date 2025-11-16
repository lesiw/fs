package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleGlob() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	files := []string{"test1.txt", "test2.txt", "data.csv"}
	for _, name := range files {
		err := fs.WriteFile(ctx, fsys, name, []byte("content"))
		if err != nil {
			log.Fatal(err)
		}
	}
	matches, err := fs.Glob(ctx, fsys, "*.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d .txt files\n", len(matches))
	// Output:
	// Found 2 .txt files
}
