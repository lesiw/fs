package fs

import (
	"context"
	"errors"

	"lesiw.io/fs/path"
)

// A LocalizeFS is a filesystem that can convert Unix-style paths
// to native paths.
//
// In most circumstances, this should be a lexical operation.
// For true path canonicalization, see [AbsFS].
//
// Operations like Open, Create, Append, and Truncate automatically call
// Localize internally when the filesystem implements LocalizeFS. The localized
// path is returned via the Path() method on the returned [ReadPathCloser] or
// [WritePathCloser].
type LocalizeFS interface {
	FS

	// Localize converts a Unix-style path to a native representation.
	//
	// Localize must be idempotent: calling Localize on an already-localized
	// path should return the same path. That is, Localize(Localize(p))
	// should equal Localize(p).
	Localize(ctx context.Context, path string) (string, error)
}

// Localize converts a path from Unix-style to native.
//
// This is typically a lexical operation.
// For canonical path representation, use [Abs].
//
// Localize may be called with an already-localized path and should return the
// same path unchanged (idempotent behavior).
//
// Requires: [LocalizeFS]
func Localize(ctx context.Context, fsys FS, path string) (string, error) {
	lfs, ok := fsys.(LocalizeFS)
	if !ok {
		return "", &PathError{
			Op:   "localize",
			Path: path,
			Err:  ErrUnsupported,
		}
	}
	return lfs.Localize(ctx, path)
}

// localizePath is an internal helper that cleans and localizes a path.
// It always returns a valid path: if localization is unsupported or fails
// with ErrUnsupported, it returns the cleaned path. Other errors are returned.
func localizePath(
	ctx context.Context, fsys FS, name string,
) (string, error) {
	name = path.Clean(name)
	lfs, ok := fsys.(LocalizeFS)
	if !ok {
		return name, nil
	}
	local, err := lfs.Localize(ctx, name)
	if err != nil && !errors.Is(err, ErrUnsupported) {
		return "", err
	}
	if err == nil {
		return local, nil
	}
	return name, nil
}
