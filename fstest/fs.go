package fstest

import (
	"context"
	"slices"
	"testing"
	"time"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

// File describes an expected file for testing.
type File struct {
	Path    string    // file or directory path
	Data    []byte    // file content (empty for directories)
	Mode    fs.Mode   // optional file mode for validation
	ModTime time.Time // optional modification time for validation
}

// TestFSOption configures TestFS behavior via functional options.
type TestFSOption func(*testFSOpts)

// testFSOpts holds configuration for TestFS.
type testFSOpts struct {
	expectedFiles []File
}

// WithFiles specifies files that must exist in the filesystem.
// When provided, TestFS validates these files exist on read-only filesystems.
//
// For writable filesystems (implementing CreateFS), WithFiles is ignored and
// tests create their own files. For read-only filesystems, WithFiles is
// required for any test that needs files, otherwise tests will fail.
//
// Example:
//
//	fstest.TestFS(ctx, t, readOnlyFS,
//	    fstest.WithFiles(
//	        fstest.File{Path: "data/file.txt", Data: []byte("content")},
//	        fstest.File{Path: "data/subdir"},
//	    ))
func WithFiles(files ...File) TestFSOption {
	return func(opts *testFSOpts) {
		opts.expectedFiles = files
	}
}

// TestFS runs a comprehensive compliance test suite on a filesystem
// implementation.
//
// Tests automatically adapt to filesystem capabilities - tests that require
// write operations will create their own test files if the filesystem is
// writable (implements CreateFS), or assume files are pre-populated if it's
// read-only.
//
// Typical usage for writable filesystem:
//
//	func TestMyFS(t *testing.T) {
//	    fsys := createBlankFS(t)
//	    ctx := context.Background()
//	    fstest.TestFS(ctx, t, fsys)  // Tests all operations
//	}
//
// Typical usage for read-only filesystem:
//
//	func TestReadOnlyFS(t *testing.T) {
//	    fsys := setupReadOnlyFS(t)  // Pre-populate with files
//	    ctx := context.Background()
//	    fstest.TestFS(ctx, t, fsys,
//	        fstest.WithFiles(
//	            fstest.File{Path: "file1.txt", Data: []byte("content")},
//	        ))
//	}
func TestFS(
	ctx context.Context, t *testing.T, fsys fs.FS, opts ...TestFSOption,
) {
	t.Helper()

	// Apply options
	var o testFSOpts
	for _, opt := range opts {
		opt(&o)
	}

	// Use provided files or default comprehensive structure
	files := o.expectedFiles
	if files == nil {
		files = defaultTestFiles()
		// Only write files if expectedFiles was not provided
		err := writeTestFiles(ctx, fsys, files)
		if err != nil {
			t.Fatalf(
				"expected writable filesystem or fstest.WithFiles: %v",
				err,
			)
		}
	}

	t.Run("Abs", func(t *testing.T) {
		testAbs(ctx, t, fsys)
	})
	t.Run("Append", func(t *testing.T) {
		testAppend(ctx, t, fsys)
	})
	t.Run("Chmod", func(t *testing.T) {
		testChmod(ctx, t, fsys)
	})
	t.Run("Chown", func(t *testing.T) {
		testChown(ctx, t, fsys)
	})
	t.Run("Chtimes", func(t *testing.T) {
		testChtimes(ctx, t, fsys)
	})
	t.Run("Create", func(t *testing.T) {
		testCreate(ctx, t, fsys)
	})
	t.Run("DirFS", func(t *testing.T) {
		testDirFS(ctx, t, fsys)
	})
	t.Run("Glob", func(t *testing.T) {
		testGlob(ctx, t, fsys, files)
	})
	t.Run("Localize", func(t *testing.T) {
		testLocalize(ctx, t, fsys)
	})
	t.Run("Mkdir", func(t *testing.T) {
		testMkdir(ctx, t, fsys)
	})
	t.Run("ReadDir", func(t *testing.T) {
		testReadDir(ctx, t, fsys, files)
	})
	t.Run("Remove", func(t *testing.T) {
		testRemove(ctx, t, fsys)
	})
	t.Run("Rename", func(t *testing.T) {
		testRename(ctx, t, fsys)
	})
	t.Run("Stat", func(t *testing.T) {
		testStat(ctx, t, fsys, files)
	})
	t.Run("Stress", func(t *testing.T) {
		testStress(ctx, t, fsys)
	})
	t.Run("Symlink", func(t *testing.T) {
		testSymlink(ctx, t, fsys)
	})
	t.Run("Temp", func(t *testing.T) {
		testTemp(ctx, t, fsys)
	})
	t.Run("Truncate", func(t *testing.T) {
		testTruncate(ctx, t, fsys)
	})
	t.Run("Walk", func(t *testing.T) {
		testWalk(ctx, t, fsys, files)
	})
	t.Run("WorkDir", func(t *testing.T) {
		testWorkDir(ctx, t, fsys)
	})
}

func normalizePath(p string) []string {
	var parts []string
	for p != "" && p != "." {
		if path.IsRoot(p) {
			break
		}
		dir, file := path.Split(p)
		if file != "" {
			parts = append([]string{file}, parts...)
		}
		if dir == "" || dir == p {
			break
		}
		p = dir
	}
	return parts
}

func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aNorm := make([][]string, len(a))
	bNorm := make([][]string, len(b))
	for i := range a {
		aNorm[i] = normalizePath(a[i])
	}
	for i := range b {
		bNorm[i] = normalizePath(b[i])
	}
	slices.SortFunc(aNorm, func(x, y []string) int {
		return slices.Compare(x, y)
	})
	slices.SortFunc(bNorm, func(x, y []string) int {
		return slices.Compare(x, y)
	})
	return slices.EqualFunc(aNorm, bNorm, slices.Equal)
}

// defaultTestFiles returns a comprehensive file structure for testing.
// Tests extract what they need from this structure.
func defaultTestFiles() []File {
	return []File{
		{Path: "a/b/c/deep.txt", Data: []byte("deep")},
		{Path: "a/b/file.txt", Data: []byte("ab")},
		{Path: "a/file.txt", Data: []byte("a")},
		{Path: "dir/nested.txt", Data: []byte("nested")},
		{Path: "dir/subdir/file.txt", Data: []byte("content")},
		{Path: "empty/.keep", Data: []byte("")},
		{Path: "file1.txt", Data: []byte("one")},
		{Path: "file2.txt", Data: []byte("two")},
		{Path: "file3.json", Data: []byte("json")},
		{Path: "x/file.txt", Data: []byte("x")},
		{Path: "x/y/file.txt", Data: []byte("xy")},
		{Path: "x/y/z/file.txt", Data: []byte("xyz")},
	}
}

// writeTestFiles writes the test file structure to a writable filesystem.
// Returns error if the filesystem doesn't support writes.
func writeTestFiles(
	ctx context.Context, fsys fs.FS, files []File,
) error {
	for _, file := range files {
		err := fs.WriteFile(ctx, fsys, file.Path, file.Data)
		if err != nil {
			return err
		}
	}
	return nil
}
