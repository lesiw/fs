package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleLstat() {
	fsys, ctx := osfs.TempFS(context.Background())
	defer fs.Close(fsys)

	err := fs.WriteFile(ctx, fsys, "target.txt", []byte("content"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.Symlink(ctx, fsys, "target.txt", "link.txt")
	if err != nil {
		log.Fatal(err)
	}
	info, err := fs.Lstat(ctx, fsys, "link.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("IsSymlink: %v\n", info.Mode()&fs.ModeSymlink != 0)
	// Output:
	// IsSymlink: true
}
