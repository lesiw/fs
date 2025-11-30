package fs

import "io"

// Pather is the interface that wraps the Path method.
//
// Path returns the native filesystem path for this resource.
type Pather interface {
	Path() string
}

// ReadPathCloser is the interface that groups the Read, Path, and Close
// methods.
type ReadPathCloser interface {
	io.Reader
	Pather
	io.Closer
}

// WritePathCloser is the interface that groups the Write, Path, and Close
// methods.
type WritePathCloser interface {
	io.Writer
	Pather
	io.Closer
}

type pather string

func (p pather) Path() string { return string(p) }

// readPathCloser composes an io.ReadCloser with a path.
func readPathCloser(rc io.ReadCloser, p string) ReadPathCloser {
	return struct {
		io.ReadCloser
		pather
	}{rc, pather(p)}
}

// writePathCloser composes an io.WriteCloser with a path.
func writePathCloser(wc io.WriteCloser, p string) WritePathCloser {
	return struct {
		io.WriteCloser
		pather
	}{wc, pather(p)}
}
