package memfs

import (
	"testing"

	"lesiw.io/fs/fstest"
)

func TestFS(t *testing.T) { fstest.TestFS(t.Context(), t, New()) }
