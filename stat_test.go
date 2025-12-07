package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleStat() {
	fsys, ctx := osfs.TempFS(context.Background())
	defer fs.Close(fsys)

	err := fs.WriteFile(ctx, fsys, "example.txt", []byte("hello"))
	if err != nil {
		log.Fatal(err)
	}
	info, err := fs.Stat(ctx, fsys, "example.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Name: %s\n", info.Name())
	fmt.Printf("Size: %d bytes\n", info.Size())
	fmt.Printf("IsDir: %v\n", info.IsDir())
	// Output:
	// Name: example.txt
	// Size: 5 bytes
	// IsDir: false
}
