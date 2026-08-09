package fstest

import (
	"context"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func testWalk(ctx context.Context, t *testing.T, fsys fs.FS, files []File) {
	var (
		_, hasWalk    = fsys.(fs.WalkFS)
		_, hasReadDir = fsys.(fs.ReadDirFS)
	)
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

func testWalkWant(files []File) (want []string) {
	seen := make(map[string]struct{})

	for _, f := range files {
		want = append(want, f.Path)

		p := f.Path
		for {
			var (
				dir   = path.Dir(p)
				_, ok = seen[dir]
			)
			if dir == "." || dir == "" || path.IsRoot(dir) || ok {
				break
			}
			want = append(want, dir)
			seen[dir] = struct{}{}
			p = dir
		}
	}

	return want
}
