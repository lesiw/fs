//go:build unix

package osfs

import (
	"context"
	"os"

	"lesiw.io/fs"
)

// Chmod implements fs.ChmodFS on Unix systems.
func (f *FS) Chmod(ctx context.Context, name string, mode fs.Mode) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

// Chown implements fs.ChownFS on Unix systems.
func (f *FS) Chown(ctx context.Context, name string, uid, gid int) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Chown(path, uid, gid)
}

// Compile-time interface checks for Unix-specific capabilities
var (
	_ fs.ChmodFS = (*FS)(nil)
	_ fs.ChownFS = (*FS)(nil)
)
