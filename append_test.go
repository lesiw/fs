package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleAppend() {
	fsys, ctx := osfs.TempFS(), context.Background()
	defer fs.Close(fsys)

	err := fs.WriteFile(ctx, fsys, "log.txt", []byte("first line\n"))
	if err != nil {
		log.Fatal(err)
	}
	f, err := fs.Append(ctx, fsys, "log.txt")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write([]byte("second line\n"))
	if err != nil {
		_ = f.Close()
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	data, err := fs.ReadFile(ctx, fsys, "log.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s", data)
	// Output:
	// first line
	// second line
}
