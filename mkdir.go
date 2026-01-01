package fs

import (
	"context"
	"errors"
)

// A MkdirFS is a file system with the Mkdir method.
type MkdirFS interface {
	FS

	// Mkdir creates a new directory.
	//
	// The directory mode is obtained from DirMode(ctx). If not set in
	// the context, the default mode 0755 is used.
	//
	// Mkdir returns an error if the directory already exists or if the
	// parent directory does not exist. Use MkdirAll to create parent
	// directories automatically.
	Mkdir(ctx context.Context, name string) error
}

// Mkdir creates a new directory.
// Analogous to: [os.Mkdir], mkdir.
//
// The directory mode is obtained from [DirMode](ctx). If not set in the
// context, the default mode 0755 is used:
//
//	ctx = fs.WithDirMode(ctx, 0700)
//	fs.Mkdir(ctx, fsys, "private")  // Creates with mode 0700
//
// Mkdir returns an error if the directory already exists or if the parent
// directory does not exist. Use [MkdirAll] to create parent directories
// automatically.
//
// Requires: [MkdirFS]
func Mkdir(ctx context.Context, fsys FS, name string) error {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return err
	}
	if mfs, ok := fsys.(MkdirFS); ok {
		if err := mfs.Mkdir(ctx, name); !errors.Is(err, ErrUnsupported) {
			return newPathError("mkdir", name, err)
		}
	}
	return &PathError{Op: "mkdir", Path: name, Err: ErrUnsupported}
}
