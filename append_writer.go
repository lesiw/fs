package fs

import "io"

// appendWriter implements append by streaming existing content followed by
// new writes. A background goroutine continuously drains the write buffer.
type appendWriter struct {
	pr   *io.PipeReader
	pw   *io.PipeWriter
	w    io.WriteCloser
	done chan error
}

// newAppendWriter creates a writer that appends to existing content.
// r may be nil if there's no existing content.
func newAppendWriter(r io.ReadCloser, w io.WriteCloser) io.WriteCloser {
	pr, pw := io.Pipe()
	aw := &appendWriter{
		pr:   pr,
		pw:   pw,
		w:    w,
		done: make(chan error),
	}

	go func() {
		var err error
		if r != nil {
			_, err = io.Copy(w, r)
			closeErr := r.Close()
			if err == nil {
				err = closeErr
			}
			if err != nil {
				pr.CloseWithError(err)
				aw.done <- err
				return
			}
		}
		_, err = io.Copy(w, pr)
		aw.done <- err
	}()

	return aw
}

func (aw *appendWriter) Write(p []byte) (n int, err error) {
	return aw.pw.Write(p)
}

func (aw *appendWriter) Close() error {
	pwErr := aw.pw.Close()
	copyErr := <-aw.done
	closeErr := aw.w.Close()

	if pwErr != nil {
		return pwErr
	}
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
