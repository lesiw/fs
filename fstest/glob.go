package fstest

import (
	"context"
	"errors"
	"slices"
	"testing"

	"lesiw.io/fs"
)

// TestGlob tests glob pattern matching.
func testGlob(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if Glob is supported (either native or via fallback)
	_, hasGlob := fsys.(fs.GlobFS)
	_, hasStat := fsys.(fs.StatFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)

	// Skip if neither native GlobFS nor all fallback requirements are present
	if !hasGlob && (!hasStat || !hasReadDir) {
		t.Skip("Glob not supported (requires GlobFS or StatFS+ReadDirFS)")
	}

	// Create test files
	globDir := "test_glob"
	files := []string{
		globDir + "/file1.txt",
		globDir + "/file2.txt",
		globDir + "/data.json",
		globDir + "/sub/nested.txt",
	}

	mkdirErr := fs.Mkdir(ctx, fsys, globDir)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for Glob test)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdir: %v", mkdirErr)
	}
	if err := fs.Mkdir(ctx, fsys, globDir+"/sub"); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}

	globData := []byte("data")
	for _, file := range files {
		if err := fs.WriteFile(ctx, fsys, file, globData); err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("write operations not supported")
			}
			t.Fatalf("write %s: %v", file, err)
		}
	}

	// Test simple wildcard
	pattern1 := globDir + "/*.txt"
	matches, globErr := fs.Glob(ctx, fsys, pattern1)
	if globErr != nil {
		t.Fatalf("glob *.txt: %v", globErr)
	}

	expected := []string{globDir + "/file1.txt", globDir + "/file2.txt"}
	slices.Sort(matches)
	slices.Sort(expected)

	if !slices.Equal(matches, expected) {
		t.Errorf("glob *.txt: got %v, expected %v", matches, expected)
	}

	// Test character class
	pattern2 := globDir + "/file[12].txt"
	matches, globErr = fs.Glob(ctx, fsys, pattern2)
	if globErr != nil {
		t.Fatalf("glob file[12].txt: %v", globErr)
	}

	if !slices.Equal(matches, expected) {
		t.Errorf("glob file[12].txt: got %v, expected %v", matches, expected)
	}

	// Test hierarchical pattern
	pattern3 := globDir + "/*/*.txt"
	matches, globErr = fs.Glob(ctx, fsys, pattern3)
	if globErr != nil {
		t.Fatalf("glob */*.txt: %v", globErr)
	}

	expected = []string{globDir + "/sub/nested.txt"}
	if !slices.Equal(matches, expected) {
		t.Errorf("glob */*.txt: got %v, expected %v", matches, expected)
	}

	// Test pattern with no matches
	pattern4 := globDir + "/*.nonexistent"
	matches, globErr = fs.Glob(ctx, fsys, pattern4)
	if globErr != nil {
		t.Fatalf("glob *.nonexistent: %v", globErr)
	}

	if len(matches) != 0 {
		t.Errorf("glob *.nonexistent: expected no matches, got %v", matches)
	}

	// Clean up
	if err := fs.RemoveAll(ctx, fsys, globDir); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}
