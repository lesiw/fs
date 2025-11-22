package fstest

import (
	"context"
	"errors"
	"path"
	"testing"

	"lesiw.io/fs"
)

func testRel(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	var relTests = []struct {
		name string
		base string
		targ string
		want string
	}{
		{"SamePath", "/a/b/c", "/a/b/c", "."},
		{"ChildPath", "/a/b", "/a/b/c", "c"},
		{"SiblingPath", "/a/b/c", "/a/b/d", "../d"},
		{"ParentPath", "/a/b/c", "/a/b", ".."},
		{"DivergentPaths", "/a/b/c", "/d/e/f", "../../../d/e/f"},
		{"BothRelative", "a/b", "a/b/c", "c"},
		{"PathsWithDotDot", "/a/b/../c", "/a/c/d", "d"},
	}

	for _, tt := range relTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fs.Rel(ctx, fsys, tt.base, tt.targ)
			if err != nil {
				if errors.Is(err, fs.ErrUnsupported) {
					t.Skip("Rel not supported")
				}
				t.Fatalf("Rel(%q, %q) err: %v", tt.base, tt.targ, err)
			}
			if got != tt.want {
				t.Errorf(
					"Rel(%q, %q) = %q, want %q",
					tt.base, tt.targ, got, tt.want,
				)
			}
		})
	}

	t.Run("MixedAbsoluteRelative", func(t *testing.T) {
		_, err := fs.Rel(ctx, fsys, "/absolute", "relative")
		if err == nil {
			t.Errorf("Rel(/absolute, relative) = nil, want error")
		}
	})

	var joinTests = []struct {
		name string
		base string
		targ string
	}{
		{"ChildPath", "/a", "/a/b"},
		{"SiblingPath", "/a/b", "/a/c"},
		{"ParentPath", "/a/b/c", "/a"},
		{"RootPath", "/", "/a/b"},
		{"DivergentPaths", "/a/b/c/d", "/x/y/z"},
	}

	for _, tt := range joinTests {
		t.Run("JoinProperty/"+tt.name, func(t *testing.T) {
			rel, err := fs.Rel(ctx, fsys, tt.base, tt.targ)
			if err != nil {
				if errors.Is(err, fs.ErrUnsupported) {
					t.Skip("Rel not supported")
				}
				t.Fatalf("Rel(%q, %q) err: %v", tt.base, tt.targ, err)
			}
			got, want := path.Join(tt.base, rel), path.Clean(tt.targ)
			if got != want {
				t.Errorf(
					"path.Join(%q, Rel(%q, %q)) = %q, want %q",
					tt.base, tt.base, tt.targ, got, want,
				)
			}
		})
	}
}
