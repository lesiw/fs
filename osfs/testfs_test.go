package osfs

import (
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/fstest"
)

func TestFS(t *testing.T) {
	fsys, ctx := NewTemp(), t.Context()
	defer fs.Close(fsys)

	fstest.TestFS(ctx, t, fsys)
}
