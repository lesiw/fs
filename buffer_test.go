package fs_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"lesiw.io/fs"
	"lesiw.io/fs/memfs"
)

func closeOnCleanup(t *testing.T, c io.Closer) {
	t.Helper()
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Errorf("cleanup Close() failed: %v", err)
		}
	})
}

func TestOpenBuffer(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	err := fs.WriteFile(ctx, fsys, "test.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := fs.OpenBuffer(ctx, fsys, "test.txt")
	closeOnCleanup(t, r)

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if got, want := string(out), "hello"; got != want {
		t.Errorf("ReadAll() = %q, want %q", got, want)
	}
}

func TestOpenBufferNotFound(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	r := fs.OpenBuffer(ctx, fsys, "nonexistent.txt")
	closeOnCleanup(t, r)

	if _, err := io.ReadAll(r); err == nil {
		t.Error("ReadAll() error = nil, want error")
	}
}

func TestOpenBufferCloseBeforeRead(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	r := fs.OpenBuffer(ctx, fsys, "test.txt")
	closeOnCleanup(t, r)

	if err := r.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestOpenBufferReadAfterClose(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	err := fs.WriteFile(ctx, fsys, "test.txt", []byte("data"))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := fs.OpenBuffer(ctx, fsys, "test.txt")
	closeOnCleanup(t, r)

	if _, err := r.Read(make([]byte, 1)); err != nil && err != io.EOF {
		t.Fatalf("Read() error = %v", err)
	}

	if err := r.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if _, err := r.Read(make([]byte, 1)); err != fs.ErrClosed {
		t.Errorf("Read() after Close() error = %v, want ErrClosed", err)
	}
}

func TestOpenBufferMultipleClose(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	err := fs.WriteFile(ctx, fsys, "test.txt", []byte("data"))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	r := fs.OpenBuffer(ctx, fsys, "test.txt")
	closeOnCleanup(t, r)

	if _, err := r.Read(make([]byte, 1)); err != nil && err != io.EOF {
		t.Fatalf("Read() error = %v", err)
	}

	if err := r.Close(); err != nil {
		t.Errorf("First Close() error = %v", err)
	}

	if err := r.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

//ignore:errcheck
func TestOpenBufferConcurrentReadClose(t *testing.T) {
	// Test concurrent Read() and Close() calls to detect race conditions.
	for range 1000 {
		ctx, fsys := context.Background(), memfs.New()
		fs.WriteFile(ctx, fsys, "test.txt", []byte("data"))

		r := fs.OpenBuffer(ctx, fsys, "test.txt")
		closeOnCleanup(t, r)

		go r.Read(make([]byte, 1))
		go r.Close()
	}
}

func TestCreateBuffer(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	w := fs.CreateBuffer(ctx, fsys, "output.txt")
	closeOnCleanup(t, w)

	if _, err := io.Copy(w, strings.NewReader("world")); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	out, err := fs.ReadFile(ctx, fsys, "output.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if got, want := string(out), "world"; got != want {
		t.Errorf("ReadFile() = %q, want %q", got, want)
	}
}

func TestCreateBufferCloseBeforeWrite(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	w := fs.CreateBuffer(ctx, fsys, "output.txt")
	closeOnCleanup(t, w)

	if err := w.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestCreateBufferWriteAfterClose(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	w := fs.CreateBuffer(ctx, fsys, "output.txt")
	closeOnCleanup(t, w)

	if _, err := w.Write([]byte("data")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if _, err := w.Write([]byte("more")); err != fs.ErrClosed {
		t.Errorf("Write() after Close() error = %v, want ErrClosed", err)
	}
}

func TestCreateBufferWriteAfterCloseUnused(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	w := fs.CreateBuffer(ctx, fsys, "output.txt")
	closeOnCleanup(t, w)

	if err := w.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	n, err := w.Write([]byte("data"))
	if err != fs.ErrClosed {
		t.Errorf("Write() after Close() error = %v, want ErrClosed", err)
	}
	if n != 0 {
		t.Errorf("Write() = %d bytes, want 0", n)
	}
}

func TestCreateBufferMultipleClose(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	w := fs.CreateBuffer(ctx, fsys, "output.txt")
	closeOnCleanup(t, w)

	if _, err := w.Write([]byte("data")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Errorf("First Close() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

//ignore:errcheck
func TestCreateBufferConcurrentWriteClose(t *testing.T) {
	// Test concurrent Write() and Close() calls to detect race conditions.
	for range 1000 {
		ctx, fsys := context.Background(), memfs.New()

		w := fs.CreateBuffer(ctx, fsys, "output.txt")
		closeOnCleanup(t, w)

		go w.Write([]byte("data"))
		go w.Close()
	}
}

func TestAppendBuffer(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	err := fs.WriteFile(ctx, fsys, "log.txt", []byte("first\n"))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	w := fs.AppendBuffer(ctx, fsys, "log.txt")
	closeOnCleanup(t, w)

	if _, err := io.Copy(w, strings.NewReader("second\n")); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	out, err := fs.ReadFile(ctx, fsys, "log.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if got, want := string(out), "first\nsecond\n"; got != want {
		t.Errorf("ReadFile() = %q, want %q", got, want)
	}
}

func TestBufferCopy(t *testing.T) {
	ctx, fsys := context.Background(), memfs.New()

	err := fs.WriteFile(ctx, fsys, "input.txt", []byte("data"))
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := io.Copy(
		fs.CreateBuffer(ctx, fsys, "output.txt"),
		fs.OpenBuffer(ctx, fsys, "input.txt"),
	); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	out, err := fs.ReadFile(ctx, fsys, "output.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if got, want := string(out), "data"; got != want {
		t.Errorf("ReadFile() = %q, want %q", got, want)
	}
}
