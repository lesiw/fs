package fstest

import (
	"context"
	"fmt"
	"testing"

	"lesiw.io/fs"
)

// TestFSOption configures TestFS behavior via functional options.
type TestFSOption func(*testFSOpts)

// testFSOpts holds configuration for TestFS.
//
// This struct can be extended in the future with additional options like:
//   - expectedContent map[string]string for validating file contents
//   - expectedInfo map[string]ExpectedFileInfo for metadata validation
//   - skipTests []string to skip specific test categories
//   - etc.
type testFSOpts struct {
	expectedFiles []string
}

// WithFiles specifies files that must exist in the filesystem.
// When provided, TestFS runs in read-only mode and validates the
// expected files exist and are readable, then skips tests that
// require write operations.
//
// This enables testing read-only filesystems where files are
// pre-populated externally.
func WithFiles(files ...string) TestFSOption {
	return func(opts *testFSOpts) {
		opts.expectedFiles = files
	}
}

// TestFS runs a comprehensive compliance test suite on a filesystem
// implementation.
//
// By default, the filesystem must be empty and writable. TestFS will
// create, modify, and delete files to test all write operations.
//
// Use WithFiles option for read-only filesystems with pre-populated files.
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
//	        fstest.WithFiles("file.txt", "dir/nested.txt"))
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

	// If expected files are provided, test read-only mode
	if len(o.expectedFiles) > 0 {
		testReadOnly(ctx, t, fsys, o.expectedFiles)
		return
	}

	// Otherwise, test all write operations on empty filesystem
	// Test basic file operations
	t.Run("File", func(t *testing.T) {
		t.Run("CreateAndRead", func(t *testing.T) {
			testCreateAndRead(ctx, t, fsys)
		})

		t.Run("WriteFile", func(t *testing.T) {
			testWriteFile(ctx, t, fsys)
		})

		t.Run("CreateTruncates", func(t *testing.T) {
			testCreateTruncates(ctx, t, fsys)
		})

		t.Run("Append", func(t *testing.T) {
			testAppend(ctx, t, fsys)
		})

		t.Run("ImplicitMkdir", func(t *testing.T) {
			testImplicitMkdir(ctx, t, fsys)
		})

		t.Run("ImplicitMkdirWithMode", func(t *testing.T) {
			testImplicitMkdirWithMode(ctx, t, fsys)
		})
	})

	// Test directory operations
	t.Run("Mkdir", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			testMkdir(ctx, t, fsys)
		})

		t.Run("All", func(t *testing.T) {
			testMkdirAll(ctx, t, fsys)
		})
	})

	t.Run("ReadDir", func(t *testing.T) {
		testReadDir(ctx, t, fsys)
	})

	// Test Walk functionality
	t.Run("Walk", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			testWalk(ctx, t, fsys)
		})

		t.Run("BreadthFirst", func(t *testing.T) {
			testWalkBreadthFirst(ctx, t, fsys)
		})

		t.Run("Lexicographic", func(t *testing.T) {
			testWalkLexicographic(ctx, t, fsys)
		})

		t.Run("Empty", func(t *testing.T) {
			testWalkEmpty(ctx, t, fsys)
		})

		t.Run("SingleLevel", func(t *testing.T) {
			testWalkSingleLevel(ctx, t, fsys)
		})

		// Test various depth limits
		for _, depth := range []int{0, 1, 2, 5} {
			t.Run(fmt.Sprintf("Depth%d", depth), func(t *testing.T) {
				testWalkDepth(ctx, t, fsys, depth)
			})
		}
	})

	// Test removal operations
	t.Run("Remove", func(t *testing.T) {
		t.Run("Basic", func(t *testing.T) {
			testRemove(ctx, t, fsys)
		})

		t.Run("All", func(t *testing.T) {
			testRemoveAll(ctx, t, fsys)
		})
	})

	// Test rename operations
	t.Run("Rename", func(t *testing.T) {
		testRename(ctx, t, fsys)
	})

	// Test Stat operations
	t.Run("Stat", func(t *testing.T) {
		testStat(ctx, t, fsys)
	})

	// Test Glob operations
	t.Run("Glob", func(t *testing.T) {
		testGlob(ctx, t, fsys)
	})

	// Test DirFS (tar) operations
	t.Run("DirFS", func(t *testing.T) {
		t.Run("OpenEmptyDir", func(t *testing.T) {
			testOpenEmptyDir(ctx, t, fsys)
		})

		t.Run("OpenDir", func(t *testing.T) {
			testOpenDir(ctx, t, fsys)
		})

		t.Run("CreateDir", func(t *testing.T) {
			testCreateDir(ctx, t, fsys)
		})
	})

	// Test metadata operations
	t.Run("Chmod", func(t *testing.T) {
		testChmod(ctx, t, fsys)
	})

	t.Run("Chown", func(t *testing.T) {
		testChown(ctx, t, fsys)
	})

	t.Run("Chtimes", func(t *testing.T) {
		testChtimes(ctx, t, fsys)
	})

	// Test file operations
	t.Run("Truncate", func(t *testing.T) {
		testTruncate(ctx, t, fsys)
	})

	// Test symlink operations
	t.Run("Symlink", func(t *testing.T) {
		t.Run("Create", func(t *testing.T) {
			testSymlink(ctx, t, fsys)
		})

		t.Run("Read", func(t *testing.T) {
			testReadlink(ctx, t, fsys)
		})
	})

	// Test temp operations
	t.Run("Temp", func(t *testing.T) {
		t.Run("File", func(t *testing.T) {
			testTempFile(ctx, t, fsys)
		})

		t.Run("Dir", func(t *testing.T) {
			testTempDir(ctx, t, fsys)
		})
	})

	// Test working directory context
	t.Run("WorkDir", func(t *testing.T) {
		testWorkDir(ctx, t, fsys)
	})

	// Test path operations
	t.Run("Abs", func(t *testing.T) {
		testAbs(ctx, t, fsys)
	})

	t.Run("Rel", func(t *testing.T) {
		testRel(ctx, t, fsys)
	})

	t.Run("Localize", func(t *testing.T) {
		TestLocalize(ctx, t, fsys)
	})

	// Stress tests combining multiple operations
	t.Run("Stress", func(t *testing.T) {
		t.Run("MixedOperations", func(t *testing.T) {
			testMixedOperations(ctx, t, fsys)
		})

		t.Run("ConcurrentReads", func(t *testing.T) {
			testConcurrentReads(ctx, t, fsys)
		})

		t.Run("ModifyAndRead", func(t *testing.T) {
			testModifyAndRead(ctx, t, fsys)
		})
	})
}
