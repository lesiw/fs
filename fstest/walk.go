// Package fstest implements support for testing filesystem implementations.
//
// The primary entry point is TestFS, which validates a filesystem
// implementation for correctness. Each test function validates a specific
// behavior independently and provides clear failure messages.
package fstest

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"testing"

	"lesiw.io/fs"
)

// pathDepth calculates the depth of a path relative to a root.
// The depth is the number of path components beyond the root.
func pathDepth(root, p string) int {
	// Clean both paths
	root = path.Clean(root)
	p = path.Clean(p)

	// If paths are equal, depth is 0
	if root == p {
		return 0
	}

	// Remove root prefix if present
	if strings.HasPrefix(p, root+"/") {
		rel := strings.TrimPrefix(p, root+"/")
		return strings.Count(rel, "/")
	}

	// If p doesn't start with root, count all components in p
	return strings.Count(p, "/")
}

// TestWalk tests basic Walk functionality with a typical directory structure.
// Creates a test structure and validates all entries are found.
func testWalk(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if Walk is supported (either native or via fallback)
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)

	// Skip if neither native WalkFS nor fallback requirement
	// (ReadDirFS) is present
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	// Create test directory structure:
	// root/
	// ├── a/
	// │   └── deep/
	// │       └── file1.txt
	// ├── b/
	// │   └── file2.txt
	// └── c.txt

	mkdirErr := fs.Mkdir(ctx, fsys, "a")
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestWalk)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdir a: %v", mkdirErr)
	}
	cleanup(ctx, t, fsys, "a")
	if err := fs.Mkdir(ctx, fsys, "a/deep"); err != nil {
		t.Fatalf("mkdir a/deep: %v", err)
	}
	file1Data := []byte("one")
	file1 := "a/deep/file1.txt"
	if err := fs.WriteFile(ctx, fsys, file1, file1Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write a/deep/file1.txt: %v", err)
	}
	if err := fs.Mkdir(ctx, fsys, "b"); err != nil {
		t.Fatalf("mkdir b: %v", err)
	}
	cleanup(ctx, t, fsys, "b")
	file2Data := []byte("two")
	file2 := "b/file2.txt"
	if err := fs.WriteFile(ctx, fsys, file2, file2Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write b/file2.txt: %v", err)
	}
	file3Data := []byte("three")
	if err := fs.WriteFile(ctx, fsys, "c.txt", file3Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write c.txt: %v", err)
	}
	cleanup(ctx, t, fsys, "c.txt")

	expected := []string{
		"a", "a/deep", "a/deep/file1.txt", "b", "b/file2.txt", "c.txt",
	}
	found := make(map[string]bool)
	var currentDepth int

	for entry, walkErr := range fs.Walk(ctx, fsys, ".", -1) {
		if walkErr != nil {
			t.Errorf("walk error: %v", walkErr)
			continue
		}

		// Normalize path
		relPath := strings.TrimPrefix(entry.Path(), ".")
		relPath = strings.TrimPrefix(relPath, "/")
		if relPath != "" {
			found[relPath] = true
		}

		// Check breadth-first: depth must be non-decreasing
		depth := pathDepth(".", entry.Path())
		if depth < currentDepth {
			t.Errorf(
				"not breadth-first: entry %q at depth %d after depth %d",
				entry.Path(), depth, currentDepth,
			)
		}
		currentDepth = depth
	}

	// Check all expected files were found
	for _, path := range expected {
		if !found[path] {
			t.Errorf("expected file not found: %s", path)
		}
	}
}

// TestWalkBreadthFirst specifically tests that breadth-first ordering
// is strictly maintained.
func testWalkBreadthFirst(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if Walk is supported (either native or via fallback)
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	// Create test structure
	mkdirErr := fs.Mkdir(ctx, fsys, "test_bfs")
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestWalkBreadthFirst)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdir test_bfs: %v", mkdirErr)
	}
	cleanup(ctx, t, fsys, "test_bfs")
	if err := fs.Mkdir(ctx, fsys, "test_bfs/a"); err != nil {
		t.Fatalf("mkdir test_bfs/a: %v", err)
	}
	if err := fs.Mkdir(ctx, fsys, "test_bfs/b"); err != nil {
		t.Fatalf("mkdir test_bfs/b: %v", err)
	}
	if err := fs.Mkdir(ctx, fsys, "test_bfs/a/deep"); err != nil {
		t.Fatalf("mkdir test_bfs/a/deep: %v", err)
	}
	testData := []byte("test")
	bfsFile := "test_bfs/a/deep/file.txt"
	if err := fs.WriteFile(ctx, fsys, bfsFile, testData); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write test_bfs/a/deep/file.txt: %v", err)
	}

	var depths []int
	var names []string

	for entry, walkErr := range fs.Walk(ctx, fsys, "test_bfs", -1) {
		if walkErr != nil {
			t.Errorf("walk error: %v", walkErr)
			continue
		}
		depths = append(depths, pathDepth("test_bfs", entry.Path()))
		names = append(names, entry.Name())
	}

	// Verify depths are non-decreasing
	for i := 1; i < len(depths); i++ {
		if depths[i] < depths[i-1] {
			t.Errorf(
				"depth decreased from %d to %d at index %d "+
					"(not breadth-first)",
				depths[i-1], depths[i], i,
			)
		}
	}

	// Verify all entries at each depth come before entries at next depth
	for i := 0; i < len(depths); i++ {
		for j := i + 1; j < len(depths); j++ {
			if depths[j] < depths[i] {
				t.Errorf(
					"entry at depth %d (index %d) after entry "+
						"at depth %d (index %d)",
					depths[j], j, depths[i], i,
				)
			}
		}
	}
}

// TestWalkLexicographic tests that entries at the same depth are in
// lexicographic order by name.
func testWalkLexicographic(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if Walk is supported (either native or via fallback)
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	// Create test structure with specific ordering
	mkdirErr := fs.Mkdir(ctx, fsys, "test_lex")
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestWalkLexicographic)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdir test_lex: %v", mkdirErr)
	}
	cleanup(ctx, t, fsys, "test_lex")
	if err := fs.Mkdir(ctx, fsys, "test_lex/z"); err != nil {
		t.Fatalf("mkdir test_lex/z: %v", err)
	}
	if err := fs.Mkdir(ctx, fsys, "test_lex/a"); err != nil {
		t.Fatalf("mkdir test_lex/a: %v", err)
	}
	if err := fs.Mkdir(ctx, fsys, "test_lex/m"); err != nil {
		t.Fatalf("mkdir test_lex/m: %v", err)
	}

	// Collect entries by depth
	byDepth := make(map[int][]string)
	for entry, walkErr := range fs.Walk(ctx, fsys, "test_lex", -1) {
		if walkErr != nil {
			t.Errorf("walk error: %v", walkErr)
			continue
		}
		depth := pathDepth("test_lex", entry.Path())
		byDepth[depth] = append(byDepth[depth], entry.Name())
	}

	// Check each depth level is sorted
	for depth, names := range byDepth {
		if depth == 0 {
			continue // Skip root
		}
		for i := 0; i+1 < len(names); i++ {
			if names[i] >= names[i+1] {
				t.Errorf(
					"entries at depth %d not in lexicographic order: %q >= %q",
					depth, names[i], names[i+1],
				)
			}
		}
	}
}

// TestWalkDepth tests that Walk correctly respects maxDepth parameter.
func testWalkDepth(
	ctx context.Context, t *testing.T, fsys fs.FS, maxDepth int,
) {
	t.Helper()

	// Check if Walk is supported (either native or via fallback)
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	// Create test structure
	testDir := fmt.Sprintf("test_depth_%d", maxDepth)
	mkdirErr := fs.Mkdir(ctx, fsys, testDir)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestWalkDepth)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdir %s: %v", testDir, mkdirErr)
	}
	cleanup(ctx, t, fsys, testDir)
	level1 := testDir + "/level1"
	if err := fs.Mkdir(ctx, fsys, level1); err != nil {
		t.Fatalf("mkdir level1: %v", err)
	}
	level2 := level1 + "/level2"
	if err := fs.Mkdir(ctx, fsys, level2); err != nil {
		t.Fatalf("mkdir level2: %v", err)
	}
	level3 := level2 + "/level3"
	if err := fs.Mkdir(ctx, fsys, level3); err != nil {
		t.Fatalf("mkdir level3: %v", err)
	}

	// Walk with depth limit
	for entry, walkErr := range fs.Walk(ctx, fsys, testDir, maxDepth) {
		if walkErr != nil {
			t.Errorf("walk error: %v", walkErr)
			continue
		}

		depth := pathDepth(testDir, entry.Path())
		// depth <= 0 means unlimited depth
		if maxDepth > 0 && depth > maxDepth {
			t.Errorf(
				"entry at depth %d exceeds maxDepth %d: %q",
				depth, maxDepth, entry.Path(),
			)
		}
	}
}

// TestWalkEmpty tests that Walk works correctly on an empty filesystem.
func testWalkEmpty(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if Walk is supported (either native or via fallback)
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	// Create empty directory
	mkdirErr := fs.Mkdir(ctx, fsys, "test_empty")
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestWalkEmpty)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdir test_empty: %v", mkdirErr)
	}
	cleanup(ctx, t, fsys, "test_empty")

	// Walk empty directory - may or may not include root depending on
	// implementation
	var count int
	for entry, walkErr := range fs.Walk(ctx, fsys, "test_empty", -1) {
		if walkErr != nil {
			t.Errorf("walk error: %v", walkErr)
			continue
		}
		count++
		// All entries should be at depth 0
		depth := pathDepth("test_empty", entry.Path())
		if depth > 0 {
			t.Errorf(
				"unexpected entry in empty dir: %q at depth %d",
				entry.Path(), depth,
			)
		}
	}

	// Empty directory should have no children (0 or 1 entry depending on
	// root inclusion)
	if count > 1 {
		t.Errorf(
			"empty directory should have at most 1 entry, got %d",
			count,
		)
	}
}

// TestWalkSingleLevel tests Walk with maxDepth=1 (equivalent to ls).
func testWalkSingleLevel(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Helper()

	// Check if Walk is supported (either native or via fallback)
	_, hasWalk := fsys.(fs.WalkFS)
	_, hasReadDir := fsys.(fs.ReadDirFS)
	if !hasWalk && !hasReadDir {
		t.Skip("Walk not supported (requires WalkFS or ReadDirFS)")
	}

	// Create test structure
	testDir := "test_single"
	mkdirErr := fs.Mkdir(ctx, fsys, testDir)
	if errors.Is(mkdirErr, fs.ErrUnsupported) {
		t.Skip("MkdirFS not supported (required for TestWalkSingleLevel)")
	}
	if mkdirErr != nil {
		t.Fatalf("mkdir test_single: %v", mkdirErr)
	}
	cleanup(ctx, t, fsys, testDir)
	file1Data := []byte("one")
	file1 := testDir + "/file1.txt"
	if err := fs.WriteFile(ctx, fsys, file1, file1Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write file1.txt: %v", err)
	}
	file2Data := []byte("two")
	file2 := testDir + "/file2.txt"
	if err := fs.WriteFile(ctx, fsys, file2, file2Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write file2.txt: %v", err)
	}
	if err := fs.Mkdir(ctx, fsys, testDir+"/subdir"); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	nestedData := []byte("nested")
	nestedFile := testDir + "/subdir/nested.txt"
	writeErr := fs.WriteFile(ctx, fsys, nestedFile, nestedData)
	if writeErr != nil {
		if errors.Is(writeErr, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("write nested.txt: %v", writeErr)
	}

	// Walk with maxDepth=1 should only see immediate children
	var names []string
	for entry, walkErr := range fs.Walk(ctx, fsys, testDir, 1) {
		if walkErr != nil {
			t.Errorf("walk error: %v", walkErr)
			continue
		}
		names = append(names, entry.Name())

		depth := pathDepth(testDir, entry.Path())
		if depth > 1 {
			t.Errorf(
				"maxDepth=1 but got entry at depth %d: %q",
				depth, entry.Path(),
			)
		}
	}

	// Should see immediate children: file1.txt, file2.txt, subdir
	// (root may or may not be included depending on implementation)
	expected := []string{"file1.txt", "file2.txt", "subdir"}
	slices.Sort(names)

	// Check that we at least got the expected children
	for _, exp := range expected {
		if !slices.Contains(names, exp) {
			t.Errorf(
				"maxDepth=0: missing expected entry %q, got %v",
				exp, names,
			)
		}
	}
}
