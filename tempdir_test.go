package fs_test

import (
	"context"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleTempDir() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	dir, err := fs.TempDir(ctx, fsys, "myapp")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := fs.RemoveAll(ctx, fsys, dir); err != nil {
			log.Fatal(err)
		}
	}()
	file, err := fs.Create(ctx, fsys, dir+"/tempfile.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	_, err = file.Write([]byte("temporary data"))
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleTempDir_emptyPrefix() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	dir, err := fs.TempDir(ctx, fsys, "")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := fs.RemoveAll(ctx, fsys, dir); err != nil {
			log.Fatal(err)
		}
	}()
}
