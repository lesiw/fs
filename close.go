package fs

import "io"

// Close closes a filesystem if it implements io.Closer.
func Close(fsys FS) error {
	if c, ok := fsys.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
