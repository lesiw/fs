//go:build unix

package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleChmod() {
	fsys, ctx := osfs.TempFS(), context.Background()
	defer fs.Close(fsys)

	err := fs.WriteFile(ctx, fsys, "perms.txt", []byte("data"))
	if err != nil {
		log.Fatal(err)
	}
	err = fs.Chmod(ctx, fsys, "perms.txt", 0444)
	if err != nil {
		log.Fatal(err)
	}
	info, err := fs.Stat(ctx, fsys, "perms.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Permissions: %o\n", info.Mode().Perm())
	// Output:
	// Permissions: 444
}
