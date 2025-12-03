package fstest

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
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

// TestCreateDir tests writing a tar stream to create a directory.
func testCreateDir(
	ctx context.Context, t *testing.T, fsys fs.FS,
) {
	if _, ok := fsys.(fs.CreateFS); !ok {
		t.Skip("CreateFS not supported")
	}

	// Create tar archive in memory
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add a file
	file1Data := []byte("created from tar")
	file1Header := &tar.Header{
		Name: "created_file.txt",
		Mode: 0644,
		Size: int64(len(file1Data)),
	}
	if err := tw.WriteHeader(file1Header); err != nil {
		t.Fatalf("WriteHeader(): %v", err)
	}
	if _, err := tw.Write(file1Data); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	// Add a directory
	dirHeader := &tar.Header{
		Name:     "created_subdir/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tw.WriteHeader(dirHeader); err != nil {
		t.Fatalf("WriteHeader() for dir: %v", err)
	}

	// Add a file in subdirectory
	file2Data := []byte("nested file from tar")
	file2Header := &tar.Header{
		Name: "created_subdir/nested.txt",
		Mode: 0644,
		Size: int64(len(file2Data)),
	}
	if err := tw.WriteHeader(file2Header); err != nil {
		t.Fatalf("WriteHeader(): %v", err)
	}
	if _, err := tw.Write(file2Data); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("Close() tar writer: %v", err)
	}

	// Extract tar to filesystem using trailing slash
	testDir := "test_createdir"

	tarWriter, err := fs.Create(ctx, fsys, testDir+"/")
	if err != nil {
		t.Fatalf("Create(%q): %v", testDir+"/", err)
	}
	cleanup(ctx, t, fsys, testDir)

	if _, err := io.Copy(tarWriter, &buf); err != nil {
		tarWriter.Close()
		t.Fatalf("Copy() tar data: %v", err)
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Close() tar writer: %v", err)
	}

	// Verify files were created
	file1Path := testDir + "/created_file.txt"
	readData, readErr := fs.ReadFile(ctx, fsys, file1Path)
	if readErr != nil {
		t.Fatalf("ReadFile(%q): %v", file1Path, readErr)
	}

	if !bytes.Equal(readData, file1Data) {
		t.Errorf(
			"ReadFile(%q) = %q, want %q",
			file1Path, readData, file1Data,
		)
	}

	// Verify subdirectory was created
	dirPath := testDir + "/created_subdir"
	info, statErr := fs.Stat(ctx, fsys, dirPath)
	if statErr != nil {
		t.Fatalf("Stat(%q): %v", dirPath, statErr)
	}

	if !info.IsDir() {
		t.Errorf("Stat(%q): IsDir() = false, want true", dirPath)
	}

	// Verify nested file
	file2Path := testDir + "/created_subdir/nested.txt"
	readData, readErr = fs.ReadFile(ctx, fsys, file2Path)
	if readErr != nil {
		t.Fatalf("ReadFile(%q): %v", file2Path, readErr)
	}

	if !bytes.Equal(readData, file2Data) {
		t.Errorf(
			"ReadFile(%q) = %q, want %q",
			file2Path, readData, file2Data,
		)
	}

	// Verify all expected files exist by walking the directory
	var foundFiles []string
	for entry, walkErr := range fs.Walk(ctx, fsys, testDir, -1) {
		if walkErr != nil {
			t.Errorf("Walk() error: %v", walkErr)
			continue
		}
		if !entry.IsDir() {
			foundFiles = append(foundFiles, entry.Name())
		}
	}

	expectedNames := []string{"created_file.txt", "nested.txt"}
	slices.Sort(foundFiles)
	slices.Sort(expectedNames)

	if !slices.Equal(foundFiles, expectedNames) {
		t.Errorf(
			"Walk() found files = %v, want %v",
			foundFiles, expectedNames,
		)
	}
}
