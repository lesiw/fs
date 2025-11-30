//go:build unix

package osfs

import (
	"context"
	"os"

	"lesiw.io/fs"
)

var _ fs.ChmodFS = (*FS)(nil)

func (f *FS) Chmod(ctx context.Context, name string, mode fs.Mode) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

var _ fs.ChownFS = (*FS)(nil)

func (f *FS) Chown(ctx context.Context, name string, uid, gid int) error {
	path, err := f.resolvePath(ctx, name)
	if err != nil {
		return err
	}
	return os.Chown(path, uid, gid)
}
