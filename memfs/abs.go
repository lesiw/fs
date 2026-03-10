package memfs

import (
	"context"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

var _ fs.AbsFS = (*memFS)(nil)

func (f *memFS) Abs(ctx context.Context, name string) (string, error) {
	if name = resolvePath(ctx, name); !path.IsAbs(name) {
		name = "/" + name
	}
	return path.Clean(name), nil
}
