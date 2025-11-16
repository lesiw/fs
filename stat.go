package fs

import "context"

// A StatFS is a file system with the Stat method.
type StatFS interface {
	FS

	// Stat returns file metadata for the named file.
	Stat(ctx context.Context, name string) (FileInfo, error)
}

// Stat returns file metadata for the named file.
// Analogous to: [io/fs.Stat], [os.Stat], stat, ls -l, 9P Tstat,
// S3 HeadObject.
func Stat(ctx context.Context, fsys FS, name string) (FileInfo, error) {
	sfs, ok := fsys.(StatFS)
	if !ok {
		return nil, &PathError{
			Op:   "stat",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	return sfs.Stat(ctx, name)
}
