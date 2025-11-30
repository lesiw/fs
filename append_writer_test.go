package fs

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestAppendWriter_NoExistingContent(t *testing.T) {
	var buf bytes.Buffer
	w := newAppendWriter(nil, nopWriteCloser{&buf})

	data := []byte("new content")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if got := buf.String(); got != "new content" {
		t.Errorf("final content = %q, want %q", got, "new content")
	}
}

func TestAppendWriter_WithExistingContent(t *testing.T) {
	existing := io.NopCloser(strings.NewReader("existing "))
	var buf bytes.Buffer
	w := newAppendWriter(existing, nopWriteCloser{&buf})

	if _, err := w.Write([]byte("appended")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	want := "existing appended"
	if got := buf.String(); got != want {
		t.Errorf("final content = %q, want %q", got, want)
	}
}

func TestAppendWriter_MultipleWrites(t *testing.T) {
	existing := io.NopCloser(strings.NewReader("start "))
	var buf bytes.Buffer
	w := newAppendWriter(existing, nopWriteCloser{&buf})

	writes := []string{"one ", "two ", "three"}
	for _, data := range writes {
		if _, err := w.Write([]byte(data)); err != nil {
			t.Fatalf("Write(%q) error = %v", data, err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	want := "start one two three"
	if got := buf.String(); got != want {
		t.Errorf("final content = %q, want %q", got, want)
	}
}

func TestAppendWriter_LargeContent(t *testing.T) {
	largeData := strings.Repeat("x", 1024*1024)
	existing := io.NopCloser(strings.NewReader(largeData))

	var buf bytes.Buffer
	w := newAppendWriter(existing, nopWriteCloser{&buf})

	appendData := "appended"
	if _, err := w.Write([]byte(appendData)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	got := buf.String()
	if !strings.HasPrefix(got, largeData) {
		t.Error("content doesn't start with existing data")
	}
	if !strings.HasSuffix(got, appendData) {
		t.Error("content doesn't end with appended data")
	}
	wantLen := len(largeData) + len(appendData)
	if len(got) != wantLen {
		t.Errorf("length = %d, want %d", len(got), wantLen)
	}
}

func TestAppendWriter_ReaderError(t *testing.T) {
	errReader := iotest.ErrReader(io.ErrUnexpectedEOF)
	existing := io.NopCloser(errReader)

	var buf bytes.Buffer
	w := newAppendWriter(existing, nopWriteCloser{&buf})

	_, err := w.Write([]byte("data"))
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Write() error = %v, want %v", err, io.ErrUnexpectedEOF)
	}
}

func TestAppendWriter_WriterError(t *testing.T) {
	existing := io.NopCloser(strings.NewReader("existing"))
	errWriter := &errorWriteCloser{writeErr: io.ErrShortWrite}
	w := newAppendWriter(existing, errWriter)

	_, err := w.Write([]byte("data"))
	if err != io.ErrShortWrite {
		t.Errorf("Write() error = %v, want %v", err, io.ErrShortWrite)
	}
}

func TestAppendWriter_CloseError(t *testing.T) {
	existing := io.NopCloser(strings.NewReader("existing"))
	errWriter := &errorWriteCloser{closeErr: io.ErrClosedPipe}
	w := newAppendWriter(existing, errWriter)

	if _, err := w.Write([]byte("data")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	err := w.Close()
	if err != io.ErrClosedPipe {
		t.Errorf("Close() error = %v, want %v", err, io.ErrClosedPipe)
	}
}

func TestAppendWriter_Streaming(t *testing.T) {
	existing := io.NopCloser(strings.NewReader("existing "))
	var buf bytes.Buffer
	w := newAppendWriter(existing, nopWriteCloser{&buf})

	if _, err := w.Write([]byte("first ")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if _, err := w.Write([]byte("second")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	want := "existing first second"
	if got := buf.String(); got != want {
		t.Errorf("final content = %q, want %q", got, want)
	}
}

// nopWriteCloser wraps an io.Writer with a no-op Close method.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// errorWriteCloser is a WriteCloser that can return errors.
type errorWriteCloser struct {
	writeErr error
	closeErr error
	buf      bytes.Buffer
}

func (w *errorWriteCloser) Write(p []byte) (n int, err error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	return w.buf.Write(p)
}

func (w *errorWriteCloser) Close() error {
	return w.closeErr
}
