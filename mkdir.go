package fs

import (
	"context"
	"errors"
	"path"
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
func Mkdir(ctx context.Context, fsys FS, name string) error {
	mfs, ok := fsys.(MkdirFS)
	if !ok {
		return &PathError{
			Op:   "mkdir",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	return mfs.Mkdir(ctx, name)
}

// MkdirAll creates a directory named name, along with any necessary parents.
// Analogous to: [os.MkdirAll], mkdir -p.
//
// The directory mode is obtained from [DirMode](ctx). If not set in the
// context, the default mode 0755 is used:
//
//	ctx = fs.WithDirMode(ctx, 0700)
//	fs.MkdirAll(ctx, fsys, "a/b/c")  // All created with mode 0700
//
// If name is already a directory, MkdirAll does nothing and returns nil.
func MkdirAll(ctx context.Context, fsys FS, name string) error {
	mfs, ok := fsys.(MkdirFS)
	if !ok {
		return &PathError{
			Op:   "mkdir",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	// Check if it already exists
	info, err := Stat(ctx, fsys, name)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return &PathError{
			Op:   "mkdir",
			Path: name,
			Err:  errors.New("not a directory"),
		}
	}

	// Try to create the directory
	err = mfs.Mkdir(ctx, name)
	if err == nil || errors.Is(err, ErrExist) {
		return nil
	}

	// Mkdir failed - try to create parent directories
	// Most commonly this is because the parent doesn't exist,
	// but we recurse regardless of the error type
	parent := path.Dir(name)
	if parent == "." || parent == name {
		return err
	}

	// Recursively create parent
	if err := MkdirAll(ctx, fsys, parent); err != nil {
		return err
	}

	// Try again (ignore ErrExist in case created by parent)
	err = mfs.Mkdir(ctx, name)
	if err == nil || errors.Is(err, ErrExist) {
		return nil
	}
	return err
}
