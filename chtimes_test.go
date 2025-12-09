package fs_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleChtimes() {
	fsys, ctx := osfs.TempFS(), context.Background()
	defer fs.Close(fsys)

	err := fs.WriteFile(ctx, fsys, "timestamps.txt", []byte("data"))
	if err != nil {
		log.Fatal(err)
	}
	mtime := time.Date(2009, 1, 1, 12, 0, 0, 0, time.UTC)
	atime := time.Now()
	err = fs.Chtimes(ctx, fsys, "timestamps.txt", atime, mtime)
	if err != nil {
		log.Fatal(err)
	}
	info, err := fs.Stat(ctx, fsys, "timestamps.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02"))
	// Output:
	// Modified: 2009-01-01
}
