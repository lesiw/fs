package fstest

import (
	"context"
	"slices"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func testGlob(ctx context.Context, t *testing.T, fsys fs.FS, files []File) {
	_, hasGlob := fsys.(fs.GlobFS)
	_, hasStat := fsys.(fs.StatFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)

	if !hasGlob && (!hasStat || !hasReadDir) {
		t.Skip("Glob not supported (requires GlobFS or StatFS+ReadDirFS)")
	}

	txtFiles := testGlobWant(files, "*.txt")
	if len(txtFiles) == 0 {
		t.Skip("no .txt files in root")
	}
	t.Run("GlobWildcard", func(t *testing.T) {
		testGlobWildcard(ctx, t, fsys, txtFiles)
	})
	nestedFiles := testGlobWant(files, "*/*.txt")
	if len(nestedFiles) > 0 {
		t.Run("GlobNested", func(t *testing.T) {
			testGlobNested(ctx, t, fsys, nestedFiles)
		})
	}
	t.Run("GlobNoMatch", func(t *testing.T) {
		testGlobNoMatch(ctx, t, fsys)
	})
}

func testGlobWildcard(
	ctx context.Context, t *testing.T, fsys fs.FS, txtFiles []string,
) {
	got, err := fs.Glob(ctx, fsys, "*.txt")
	if err != nil {
		t.Fatalf("Glob(\"*.txt\") = %v", err)
	}

	if !pathsEqual(got, txtFiles) {
		t.Errorf("Glob(\"*.txt\") = %v, want %v", got, txtFiles)
	}
}

func testGlobNested(
	ctx context.Context, t *testing.T, fsys fs.FS, nestedFiles []string,
) {
	got, err := fs.Glob(ctx, fsys, "*/*.txt")
	if err != nil {
		t.Fatalf("Glob(\"*/*.txt\") = %v", err)
	}

	if !pathsEqual(got, nestedFiles) {
		t.Errorf("Glob(\"*/*.txt\") = %v, want %v", got, nestedFiles)
	}
}

func testGlobNoMatch(ctx context.Context, t *testing.T, fsys fs.FS) {
	got, err := fs.Glob(ctx, fsys, "*.nonexistent")
	if err != nil {
		t.Fatalf("Glob(\"*.nonexistent\") = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Glob(\"*.nonexistent\") = %v, want []", got)
	}
}

func testGlobWant(files []File, pattern string) []string {
	var want []string

	patternHasDir := path.Dir(pattern) != "."

	for _, f := range files {
		matched, err := path.Match(pattern, f.Path)
		if err == nil && matched {
			want = append(want, f.Path)
		}

		if !patternHasDir {
			if path.Dir(f.Path) == "." {
				matched, err := path.Match(pattern, f.Path)
				if err == nil && matched {
					if !slices.Contains(want, f.Path) {
						want = append(want, f.Path)
					}
				}
			}
		}
	}

	return want
}
