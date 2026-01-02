package fs

import (
	"context"
	"io"
	"sync"
)

// OpenBuffer returns a lazy-executing reader for the file at name.
// The file is opened on first Read(), not when OpenBuffer is called.
//
// Example:
//
//	io.Copy(dst, fs.OpenBuffer(ctx, fsys, "input.txt"))
func OpenBuffer(ctx context.Context, fsys FS, name string) io.ReadCloser {
	return &bufferReader{ctx: ctx, fsys: fsys, name: name}
}

type bufferReader struct {
	sync.Mutex
	ctx  context.Context
	fsys FS
	name string

	r       io.ReadCloser
	err     error
	started bool
	closed  bool
}

func (br *bufferReader) init() {
	br.r, br.err = Open(br.ctx, br.fsys, br.name)
}

func (br *bufferReader) Read(p []byte) (n int, err error) {
	br.Lock()
	if br.closed {
		br.Unlock()
		return 0, ErrClosed
	}
	if !br.started {
		br.started = true
		br.init()
	}
	br.Unlock()

	if br.err != nil {
		return 0, br.err
	}
	return br.r.Read(p)
}

func (br *bufferReader) Close() error {
	br.Lock()
	if br.closed {
		br.Unlock()
		return nil
	}
	br.closed = true
	started := br.started
	br.Unlock()

	if started && br.r != nil {
		return br.r.Close()
	}
	return nil
}

// CreateBuffer returns a lazy-executing writer for the file at name.
// The file is created on first Write(), not when CreateBuffer is called.
//
// Example:
//
//	io.Copy(fs.CreateBuffer(ctx, fsys, "output.txt"), src)
func CreateBuffer(
	ctx context.Context, fsys FS, name string,
) io.WriteCloser {
	return &bufferWriter{ctx: ctx, fsys: fsys, name: name, create: true}
}

// AppendBuffer returns a lazy-executing writer for appending to the file at
// name. The file is opened for appending on first Write(), not when
// AppendBuffer is called.
//
// Example:
//
//	io.Copy(fs.AppendBuffer(ctx, fsys, "log.txt"), src)
func AppendBuffer(
	ctx context.Context, fsys FS, name string,
) io.WriteCloser {
	return &bufferWriter{ctx: ctx, fsys: fsys, name: name, create: false}
}

type bufferWriter struct {
	sync.Mutex
	ctx    context.Context
	fsys   FS
	name   string
	create bool // true for Create, false for Append

	w       io.WriteCloser
	err     error
	started bool
	closed  bool
}

func (bw *bufferWriter) init() {
	if bw.create {
		cfs, ok := bw.fsys.(CreateFS)
		if !ok {
			bw.err = ErrUnsupported
			return
		}
		bw.w, bw.err = cfs.Create(bw.ctx, bw.name)
	} else {
		afs, ok := bw.fsys.(AppendFS)
		if !ok {
			bw.err = ErrUnsupported
			return
		}
		bw.w, bw.err = afs.Append(bw.ctx, bw.name)
	}
}

func (bw *bufferWriter) Write(p []byte) (n int, err error) {
	bw.Lock()
	if bw.closed {
		bw.Unlock()
		return 0, ErrClosed
	}
	if !bw.started {
		bw.started = true
		bw.init()
	}
	bw.Unlock()

	if bw.err != nil {
		return 0, bw.err
	}
	return bw.w.Write(p)
}

func (bw *bufferWriter) Close() error {
	bw.Lock()
	if bw.closed {
		bw.Unlock()
		return nil
	}
	bw.closed = true
	started := bw.started
	bw.Unlock()

	if started && bw.w != nil {
		return bw.w.Close()
	}
	return nil
}

// ReadFrom implements io.ReaderFrom for optimized copying that auto-closes
// the writer when the source reaches EOF.
// This allows io.Copy to automatically close the writer in pipeline stages.
func (bw *bufferWriter) ReadFrom(src io.Reader) (n int64, err error) {
	bw.Lock()
	if bw.closed {
		bw.Unlock()
		return 0, ErrClosed
	}
	if !bw.started {
		bw.started = true
		bw.init()
	}
	bw.Unlock()

	if bw.err != nil {
		return 0, bw.err
	}

	n, err = io.Copy(bw.w, src)

	closeErr := bw.Close()
	if err == nil {
		err = closeErr
	}

	return n, err
}
