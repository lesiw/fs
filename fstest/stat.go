package fstest

import (
	"context"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/path"
)

func testStat(ctx context.Context, t *testing.T, fsys fs.FS, files []File) {
	sfs, ok := fsys.(fs.StatFS)
	if !ok {
		t.Skip("StatFS not supported")
	}

	file, dir := testStatWant(files)

	if file != nil {
		t.Run("StatFile", func(t *testing.T) {
			info, err := sfs.Stat(ctx, file.Path)
			if err != nil {
				t.Fatalf("Stat(%q) = %v", file.Path, err)
			}

			if info.IsDir() {
				t.Errorf("Stat(%q): IsDir() = true, want false", file.Path)
			}

			if got, want := info.Name(), path.Base(file.Path); got != want {
				t.Errorf(
					"Stat(%q): Name() = %q, want %q",
					file.Path, got, want,
				)
			}

			size := int64(len(file.Data))
			if got, want := info.Size(), size; got != want {
				t.Errorf(
					"Stat(%q): Size() = %d, want %d",
					file.Path, got, want,
				)
			}

			if file.Mode != 0 {
				if got := info.Mode().Perm(); got != file.Mode.Perm() {
					t.Errorf(
						"Stat(%q): Mode().Perm() = %o, want %o",
						file.Path, got, file.Mode.Perm(),
					)
				}
			}

			if !file.ModTime.IsZero() {
				if got := info.ModTime(); !got.Equal(file.ModTime) {
					t.Errorf(
						"Stat(%q): ModTime() = %v, want %v",
						file.Path, got, file.ModTime,
					)
				}
			}
		})
	}

	if dir != "" {
		t.Run("StatDirectory", func(t *testing.T) {
			info, err := sfs.Stat(ctx, dir)
			if err != nil {
				t.Fatalf("Stat(%q) = %v", dir, err)
			}

			if !info.IsDir() {
				t.Errorf("Stat(%q): IsDir() = false, want true", dir)
			}

			if got, want := info.Name(), path.Base(dir); got != want {
				t.Errorf("Stat(%q): Name() = %q, want %q", dir, got, want)
			}
		})
	}

	t.Run("StatNonexistent", func(t *testing.T) {
		_, err := sfs.Stat(ctx, "test_stat_nonexistent")
		if err == nil {
			t.Errorf("Stat(nonexistent) = nil, want error")
		}
	})
}

func testStatWant(files []File) (*File, string) {
	var file *File
	var dir string

	for i := range files {
		if file == nil {
			file = &files[i]
		}
		if d := path.Dir(files[i].Path); d != "." && dir == "" {
			dir = d
		}
		if file != nil && dir != "" {
			break
		}
	}

	return file, dir
}
