package fstest

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func testDirFS(ctx context.Context, t *testing.T, fsys fs.FS) {
	t.Run("OpenEmptyDir", func(t *testing.T) {
		testOpenEmptyDir(ctx, t, fsys)
	})
	t.Run("OpenDir", func(t *testing.T) {
		testOpenDir(ctx, t, fsys)
	})
	t.Run("CreateDir", func(t *testing.T) {
		testCreateDir(ctx, t, fsys)
	})
}

// testOpenEmptyDir tests reading an empty directory as a tar stream.
// This requires MkdirFS support to create the empty directory.
func testOpenEmptyDir(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	if _, ok := fsys.(fs.StatFS); !ok {
		t.Skip("StatFS not supported - cannot detect directories")
	}

	// Create test directory structure
	testDir := "test_opendir"
	if err := fs.MkdirAll(ctx, fsys, testDir+"/subdir"); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("MkdirFS not supported (required for empty directory test)")
		}
		t.Fatalf("MkdirAll(): %v", err)
	}
	cleanup(ctx, t, fsys, testDir)

	file1Data := []byte("file one")
	file1 := testDir + "/file1.txt"
	if err := fs.WriteFile(ctx, fsys, file1, file1Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", file1, err)
	}

	file2Data := []byte("file two")
	file2 := testDir + "/subdir/file2.txt"
	if err := fs.WriteFile(ctx, fsys, file2, file2Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", file2, err)
	}

	// Open directory as tar stream using trailing slash
	tarReader, err := fs.Open(ctx, fsys, testDir+"/")
	if err != nil {
		t.Fatalf("Open(%q): %v", testDir+"/", err)
	}
	defer tarReader.Close()

	// Read and verify tar contents
	tr := tar.NewReader(tarReader)
	foundFiles := make(map[string][]byte)

	for {
		hdr, tarErr := tr.Next()
		if tarErr == io.EOF {
			break
		}
		if tarErr != nil {
			t.Fatalf("tar.Next(): %v", tarErr)
		}

		// Read file contents
		if !hdr.FileInfo().IsDir() {
			data, readErr := io.ReadAll(tr)
			if readErr != nil {
				t.Fatalf("ReadAll(%q): %v", hdr.Name, readErr)
			}
			foundFiles[path.Clean(hdr.Name)] = data
		}
	}

	// Verify expected files were found
	expectedFiles := map[string][]byte{
		"file1.txt":        file1Data,
		"subdir/file2.txt": file2Data,
	}

	for name, expectedData := range expectedFiles {
		var data []byte
		var found bool
		for foundPath, foundData := range foundFiles {
			if pathsEqual([]string{foundPath}, []string{name}) {
				data = foundData
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tar archive missing file: %q", name)
			continue
		}
		if !bytes.Equal(data, expectedData) {
			t.Errorf(
				"tar file %q content = %q, want %q",
				name, data, expectedData,
			)
		}
	}

}

// testOpenDir tests reading a directory created by files.
// This works on all filesystems, including those with virtual directories.
func testOpenDir(ctx context.Context, t *testing.T, fsys fs.FS) {
	// Create files in nested directories
	// (directories created explicitly or implicitly depending on filesystem)
	testDir := "test_opendir_files"

	file1Data := []byte("file one")
	file1 := testDir + "/file1.txt"
	if err := fs.WriteFile(ctx, fsys, file1, file1Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", file1, err)
	}
	cleanup(ctx, t, fsys, testDir)

	file2Data := []byte("file two")
	file2 := testDir + "/subdir/file2.txt"
	if err := fs.WriteFile(ctx, fsys, file2, file2Data); err != nil {
		if errors.Is(err, fs.ErrUnsupported) {
			t.Skip("write operations not supported")
		}
		t.Fatalf("WriteFile(%q): %v", file2, err)
	}

	// Open directory as tar stream using trailing slash
	tarReader, err := fs.Open(ctx, fsys, testDir+"/")
	if err != nil {
		t.Fatalf("Open(%q): %v", testDir+"/", err)
	}
	defer tarReader.Close()

	// Read and verify tar contents
	tr := tar.NewReader(tarReader)
	foundFiles := make(map[string][]byte)

	for {
		hdr, tarErr := tr.Next()
		if tarErr == io.EOF {
			break
		}
		if tarErr != nil {
			t.Fatalf("tar.Next(): %v", tarErr)
		}

		// Read file contents
		if !hdr.FileInfo().IsDir() {
			data, readErr := io.ReadAll(tr)
			if readErr != nil {
				t.Fatalf("ReadAll(%q): %v", hdr.Name, readErr)
			}
			foundFiles[path.Clean(hdr.Name)] = data
		}
	}

	// Verify expected files were found
	expectedFiles := map[string][]byte{
		"file1.txt":        file1Data,
		"subdir/file2.txt": file2Data,
	}

	for name, expectedData := range expectedFiles {
		var data []byte
		var found bool
		for foundPath, foundData := range foundFiles {
			if pathsEqual([]string{foundPath}, []string{name}) {
				data = foundData
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tar archive missing file: %q", name)
			continue
		}
		if !bytes.Equal(data, expectedData) {
			t.Errorf(
				"tar file %q content = %q, want %q",
				name, data, expectedData,
			)
		}
	}

}

// testCreateDir tests writing tar streams to create directories.
func testCreateDir(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	if _, ok := fsys.(fs.CreateFS); !ok {
		t.Skip("CreateFS not supported")
	}

	type entry struct {
		header *tar.Header
		data   []byte
	}

	tests := []struct {
		name    string
		entries []entry
		padding int // trailing zero bytes after end-of-archive
		files   map[string][]byte
	}{{
		name: "basic",
		entries: []entry{{
			&tar.Header{Name: "file.txt", Mode: 0644},
			[]byte("created from tar"),
		}, {
			&tar.Header{
				Name:     "subdir/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			},
			nil,
		}, {
			&tar.Header{Name: "subdir/nested.txt", Mode: 0644},
			[]byte("nested file from tar"),
		}},
		files: map[string][]byte{
			"file.txt":          []byte("created from tar"),
			"subdir/nested.txt": []byte("nested file from tar"),
		},
	}, {
		name: "padded",
		entries: []entry{{
			&tar.Header{Name: "file.txt", Mode: 0644},
			[]byte("padded tar content"),
		}},
		padding: 8192,
		files: map[string][]byte{
			"file.txt": []byte("padded tar content"),
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			tw := tar.NewWriter(&buf)

			for _, e := range tt.entries {
				e.header.Size = int64(len(e.data))
				if err := tw.WriteHeader(e.header); err != nil {
					t.Fatalf("WriteHeader(%q): %v", e.header.Name, err)
				}
				if len(e.data) > 0 {
					if _, err := tw.Write(e.data); err != nil {
						t.Fatalf("Write(%q): %v", e.header.Name, err)
					}
				}
			}
			if err := tw.Close(); err != nil {
				t.Fatalf("Close() tar writer: %v", err)
			}
			if tt.padding > 0 {
				buf.Write(make([]byte, tt.padding))
			}

			testDir := "test_createdir_" + tt.name
			w, err := fs.Create(ctx, fsys, testDir+"/")
			if err != nil {
				t.Fatalf("Create(%q): %v", testDir+"/", err)
			}
			cleanup(ctx, t, fsys, testDir)

			if _, err := io.Copy(w, &buf); err != nil {
				w.Close()
				t.Fatalf("Copy(): %v", err)
			}
			if err := w.Close(); err != nil {
				t.Fatalf("Close(): %v", err)
			}

			for name, want := range tt.files {
				got, err := fs.ReadFile(ctx, fsys, testDir+"/"+name)
				if err != nil {
					t.Fatalf("ReadFile(%q): %v", name, err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf(
						"ReadFile(%q) = %q, want %q",
						name, got, want,
					)
				}
			}
		})
	}
}
