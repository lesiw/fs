package fs_test

import (
	"context"
	"fmt"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleWriteFile() {
	ctx := context.Background()
	fsys, err := osfs.New("")
	if err != nil {
		log.Fatal(err)
	}
	defer fsys.Close()

	data := []byte("Hello, filesystem!")
	err = fs.WriteFile(ctx, fsys, "output.txt", data)
	if err != nil {
		log.Fatal(err)
	}
	readData, err := fs.ReadFile(ctx, fsys, "output.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", readData)
	// Output:
	// Hello, filesystem!
}
