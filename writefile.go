package fs

import "context"

// WriteFile writes data to the named file in the filesystem.
// It creates the file or truncates it if it already exists.
// The file is created with mode 0644 (or the mode specified via
// WithFileMode).
//
// Like os.WriteFile, this always truncates existing files to zero length
// before writing.
//
// If the parent directory does not exist and the filesystem implements
// MkdirFS, WriteFile automatically creates the parent directories with
// mode 0755 (or the mode specified via WithDirMode).
//
// This is analogous to os.WriteFile and io/fs.ReadFile.
func WriteFile(
	ctx context.Context, fsys FS, name string, data []byte,
) error {
	f, err := Create(ctx, fsys, name)
	if err != nil {
		return err
	}

	_, writeErr := f.Write(data)
	closeErr := f.Close()

	if writeErr != nil {
		return &PathError{Op: "write", Path: name, Err: writeErr}
	}
	if closeErr != nil {
		return &PathError{Op: "close", Path: name, Err: closeErr}
	}
	return nil
}
