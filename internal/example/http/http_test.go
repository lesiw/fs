package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lesiw.io/fs/fstest"
)

func TestHTTPFS(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files for read-only filesystem
	testFiles := []fstest.File{
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

	for _, f := range testFiles {
		fullPath := filepath.Join(tmpDir, filepath.FromSlash(f.Path))
		// Create parent directory
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, f.Data, 0644); err != nil {
			t.Fatalf("write %s: %v", fullPath, err)
		}
	}

	// Start HTTP file server in background
	server := httptest.NewServer(http.FileServer(http.Dir(tmpDir)))
	defer server.Close()

	// Create HTTP filesystem pointing to test server
	fsys := New(server.URL)

	ctx := t.Context()

	// Run the fstest suite with WithFiles for read-only filesystem
	fstest.TestFS(ctx, t, fsys, fstest.WithFiles(testFiles...))
}
