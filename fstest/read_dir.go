package fstest

import (
	"context"
	"errors"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func testReadDir(ctx context.Context, t *testing.T, fsys fs.FS, files []File) {
	t.Run("ReadDirDot", func(t *testing.T) {
		testReadDirDot(ctx, t, fsys)
	})
	t.Run("ReadDirCurrent", func(t *testing.T) {
		testReadDirCurrent(ctx, t, fsys, files)
	})
}

func testReadDirDot(ctx context.Context, t *testing.T, fsys fs.FS) {
	// Reading "." should always work on a filesystem, even if it's empty.
	for _, err := range fs.ReadDir(ctx, fsys, ".") {
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("ReadDirFS not supported")
			}
			t.Fatalf(`ReadDir(".") failed: %v`, err)
		}
	}
}

func testReadDirCurrent(
	ctx context.Context, t *testing.T, fsys fs.FS, files []File,
) {
	want := testReadDirWant(files)

	var names []string
	var entries []fs.DirEntry
	for e, err := range fs.ReadDir(ctx, fsys, ".") {
		if err != nil {
			if errors.Is(err, fs.ErrUnsupported) {
				t.Skip("ReadDirFS not supported")
			}
			t.Fatalf("ReadDir(\".\") iteration: %v", err)
		}
		names = append(names, e.Name())
		entries = append(entries, e)
	}

	for _, e := range entries {
		w, ok := want[e.Name()]
		if !ok {
			t.Errorf("ReadDir(\".\") unexpected %q", e.Name())
			continue
		}

		if e.IsDir() != w.isDir {
			t.Errorf(
				"ReadDir(\".\") %q: IsDir() = %v, want %v",
				e.Name(), e.IsDir(), w.isDir,
			)
		}

		if !w.isDir {
			info, err := e.Info()
			if err != nil {
				t.Errorf("ReadDir(\".\") %q: Info() = %v", e.Name(), err)
				continue
			}

			if info.Size() != w.size {
				t.Errorf(
					"ReadDir(\".\") %q: Size() = %d, want %d",
					e.Name(), info.Size(), w.size,
				)
			}
		}
	}

	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}
	for name := range want {
		if !found[name] {
			t.Errorf("ReadDir(\".\") missing %q", name)
		}
	}
}

type readDirEntry struct {
	isDir bool
	size  int64
}

func testReadDirWant(files []File) map[string]readDirEntry {
	want := make(map[string]readDirEntry)

	for _, f := range files {
		// Check if path has directory component
		dir := path.Dir(f.Path)
		if dir != "." && dir != "" {
			// Nested file - add directory to want
			// Get the top-level directory name
			name := f.Path
			for {
				parent := path.Dir(name)
				if parent == "." || parent == "" || path.IsRoot(parent) {
					break
				}
				name = parent
			}
			if _, exists := want[name]; !exists {
				want[name] = readDirEntry{isDir: true}
			}
		} else {
			// Root-level file
			want[f.Path] = readDirEntry{
				isDir: false,
				size:  int64(len(f.Data)),
			}
		}
	}

	return want
}
