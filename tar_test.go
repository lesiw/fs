package fs_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"lesiw.io/fs"
	"lesiw.io/fs/osfs"
)

func ExampleOpen_directory() {
	fsys, ctx := osfs.TempFS(context.Background())
	defer fs.Close(fsys)

	err := fs.MkdirAll(ctx, fsys, "project/src")
	if err != nil {
		log.Fatal(err)
	}
	err = fs.WriteFile(ctx, fsys, "project/README.md", []byte("# Project"))
	if err != nil {
		log.Fatal(err)
	}
	data := []byte("package main")
	err = fs.WriteFile(ctx, fsys, "project/src/main.go", data)
	if err != nil {
		log.Fatal(err)
	}
	tarReader, err := fs.Open(ctx, fsys, "project/")
	if err != nil {
		log.Fatal(err)
	}
	defer tarReader.Close()
	var buf bytes.Buffer
	n, err := buf.ReadFrom(tarReader)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Created tar archive: %d bytes\n", n)
	// Output:
	// Created tar archive: 3584 bytes
}

func ExampleCreate_directory() {
	fsys, ctx := osfs.TempFS(context.Background())
	defer fs.Close(fsys)

	err := fs.MkdirAll(ctx, fsys, "source/data")
	if err != nil {
		log.Fatal(err)
	}
	err = fs.WriteFile(ctx, fsys, "source/data/file.txt", []byte("content"))
	if err != nil {
		log.Fatal(err)
	}
	err = func() error {
		tr, err := fs.Open(ctx, fsys, "source/")
		if err != nil {
			return err
		}
		defer tr.Close()
		tw, err := fs.Create(ctx, fsys, "dest/")
		if err != nil {
			return err
		}
		defer tw.Close()
		_, err = io.Copy(tw, tr)
		return err
	}()
	if err != nil {
		log.Fatal(err)
	}
	data, err := fs.ReadFile(ctx, fsys, "dest/data/file.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", data)
	// Output:
	// content
}
