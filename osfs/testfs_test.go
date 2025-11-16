package osfs

import (
	"testing"

	"lesiw.io/fs/fstest"
)

func TestFS(t *testing.T) {
	fsys, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()

	fstest.TestFS(t.Context(), t, fsys)
}
