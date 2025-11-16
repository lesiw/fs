package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"lesiw.io/fs/fstest"
)

func TestHTTPFS(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files that HTTP server will serve
	testFiles := []string{"test.txt", "dir/nested.txt", "another.txt"}
	for _, path := range testFiles {
		fullPath := tmpDir + "/" + path
		if path == "dir/nested.txt" {
			if err := os.MkdirAll(tmpDir+"/dir", 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
		}
		content := []byte("hello from " + path)
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Start HTTP file server in background
	server := httptest.NewServer(http.FileServer(http.Dir(tmpDir)))
	defer server.Close()

	// Create HTTP filesystem pointing to test server
	fsys := New(server.URL)

	ctx := t.Context()

	// Run the fstest suite with expected files (read-only mode)
	fstest.TestFS(ctx, t, fsys, fstest.WithFiles(testFiles...))
}
