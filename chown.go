package fs

import "context"

// A ChownFS is a file system with the Chown method.
type ChownFS interface {
	FS

	// Chown changes the numeric uid and gid of the named file.
	// This is typically a Unix-specific operation.
	Chown(ctx context.Context, name string, uid, gid int) error
}

// Chown changes the numeric uid and gid of the named file.
// Analogous to: [os.Chown], [os.Lchown], chown, 9P Twstat.
// This is typically a Unix-specific operation.
func Chown(ctx context.Context, fsys FS, name string, uid, gid int) error {
	cfs, ok := fsys.(ChownFS)
	if !ok {
		return &PathError{
			Op:   "chown",
			Path: name,
			Err:  ErrUnsupported,
		}
	}

	return cfs.Chown(ctx, name, uid, gid)
}
