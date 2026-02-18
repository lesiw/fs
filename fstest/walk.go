package fstest

import (
	"context"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func testWalk(ctx context.Context, t *testing.T, fsys fs.FS, files []File) {
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	want := testWalkWant(files)
	var found []string

	for e, err := range fs.Walk(ctx, fsys, ".", -1) {
		if err != nil {
			t.Errorf("Walk(\".\") iteration: %v", err)
			continue
		}

		p := e.Path()
		if p != "." {
			found = append(found, p)
		}
	}

	if !pathsEqual(found, want) {
		t.Errorf("Walk(\".\") = %v, want %v", found, want)
	}
}

func testWalkWant(files []File) []string {
	var want []string
	seen := make(map[string]bool)

	for _, f := range files {
		want = append(want, f.Path)

		p := f.Path
		for {
			dir := path.Dir(p)
			if dir == "." || dir == "" || path.IsRoot(dir) || seen[dir] {
				break
			}
			want = append(want, dir)
			seen[dir] = true
			p = dir
		}
	}

	return want
}
