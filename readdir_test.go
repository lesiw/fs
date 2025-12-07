package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleWalk_singleLevel() {
	fsys, ctx := osfs.TempFS(context.Background())
	defer fs.Close(fsys)

	err := fs.Mkdir(ctx, fsys, "testdir")
	if err != nil {
		log.Fatal(err)
	}
	err = fs.WriteFile(ctx, fsys, "testdir/file1.txt", []byte("one"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.WriteFile(ctx, fsys, "testdir/file2.txt", []byte("two"))
	if err != nil {
		log.Fatal(err)
	}
	var entries []fs.DirEntry
	for entry, err := range fs.Walk(ctx, fsys, "testdir", 0) {
		if err != nil {
			log.Fatal(err)
		}
		entries = append(entries, entry)
	}
	fmt.Printf("Found %d entries:\n", len(entries))
	for _, entry := range entries {
		fmt.Printf("- %s (dir: %v)\n", entry.Name(), entry.IsDir())
	}
	// Output:
	// Found 2 entries:
	// - file1.txt (dir: false)
	// - file2.txt (dir: false)
}
