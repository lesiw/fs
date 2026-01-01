package fstest

import (
	"context"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func pathDepth(root, p string) int {
	root = path.Clean(root)
	p = path.Clean(p)

	if root == p {
		return 0
	}

	depth := 0
	current := p
	for {
		dir := path.Dir(current)
		if dir == "" || dir == "." || dir == root || path.IsRoot(dir) {
			break
		}
		if dir == current {
			break
		}
		depth++
		current = dir
	}

	return depth
}

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

func testWalkDepth(
	ctx context.Context,
	t *testing.T,
	fsys fs.FS,
	files []File,
	maxDepth int,
) {
	t.Helper()

	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	for e, err := range fs.Walk(ctx, fsys, ".", maxDepth) {
		if err != nil {
			t.Errorf("Walk(\".\", %d) iteration: %v", maxDepth, err)
			continue
		}

		depth := pathDepth(".", e.Path())
		if maxDepth > 0 && depth > maxDepth {
			t.Errorf(
				"Walk(\".\", %d) %q at depth %d",
				maxDepth, e.Path(), depth,
			)
		}
	}
}

func testWalkEmpty(
	ctx context.Context, t *testing.T, fsys fs.FS, files []File,
) {
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	dir := testWalkEmptyDir(files)
	if dir == "" {
		t.Skip("no empty directory in files")
	}

	var n int
	for e, err := range fs.Walk(ctx, fsys, dir, -1) {
		if err != nil {
			t.Errorf("Walk(%q) iteration: %v", dir, err)
			continue
		}
		n++
		depth := pathDepth(dir, e.Path())
		if depth > 0 {
			t.Errorf("Walk(%q) %q at depth %d", dir, e.Path(), depth)
		}
	}

	if n > 1 {
		t.Errorf("Walk(%q) = %d entries, want <= 1", dir, n)
	}
}

func testWalkEmptyDir(files []File) string {
	dirs := make(map[string]bool)
	hasContent := make(map[string]bool)

	for _, f := range files {
		p := f.Path
		for {
			dir := path.Dir(p)
			if dir == "." || dir == "" || path.IsRoot(dir) {
				break
			}
			dirs[dir] = true
			hasContent[dir] = true
			p = dir
		}
	}

	for dir := range dirs {
		if !hasContent[dir] {
			return dir
		}
	}
	return ""
}

func testWalkSingleLevel(
	ctx context.Context, t *testing.T, fsys fs.FS, files []File,
) {
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	want := testWalkSingleLevelWant(files)
	if len(want) == 0 {
		t.Skip("no multi-level directory in files")
	}

	var found []string
	for e, err := range fs.Walk(ctx, fsys, ".", 1) {
		if err != nil {
			t.Errorf("Walk(\".\", 1) iteration: %v", err)
			continue
		}

		depth := pathDepth(".", e.Path())
		if depth > 1 {
			t.Errorf("Walk(\".\", 1) %q at depth %d", e.Path(), depth)
		}

		p := e.Path()
		if p != "." {
			found = append(found, p)
		}
	}

	if !pathsEqual(found, want) {
		t.Errorf("Walk(\".\", 1) = %v, want %v", found, want)
	}
}

func testWalkSingleLevelWant(files []File) []string {
	var want []string

	for _, f := range files {
		p := f.Path
		for {
			dir := path.Dir(p)
			if dir == "." || dir == "" || path.IsRoot(dir) {
				want = append(want, p)
				break
			}
			p = dir
		}
	}

	return want
}
