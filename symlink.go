package fs

import "context"

// A SymlinkFS is a file system with the Symlink method.
type SymlinkFS interface {
	FS

	// Symlink creates newname as a symbolic link to oldname.
	Symlink(ctx context.Context, oldname, newname string) error
}

// A ReadLinkFS is a file system with the ReadLink and Lstat methods.
type ReadLinkFS interface {
	FS
	// ReadLink returns the destination of the named symbolic link.
	// If the link destination is relative, ReadLink returns the relative
	// path without resolving it to an absolute one.
	ReadLink(ctx context.Context, name string) (string, error)

	// Lstat returns FileInfo describing the named file.
	// If the file is a symbolic link, the returned FileInfo
	// describes the symbolic link. Lstat makes no attempt to follow
	// the link.
	Lstat(ctx context.Context, name string) (FileInfo, error)
}

// Symlink creates newname as a symbolic link to oldname.
// Analogous to: [os.Symlink], ln -s, 9P2000.u Tsymlink.
//
// Requires: [SymlinkFS]
func Symlink(
	ctx context.Context, fsys FS, oldname, newname string,
) error {
	var err error
	if oldname, err = localizePath(ctx, fsys, oldname); err != nil {
		return err
	}
	if newname, err = localizePath(ctx, fsys, newname); err != nil {
		return err
	}
	if sfs, ok := fsys.(SymlinkFS); ok {
		return sfs.Symlink(ctx, oldname, newname)
	}
	return &PathError{
		Op:   "symlink",
		Path: newname,
		Err:  ErrUnsupported,
	}
}

// ReadLink returns the destination of the named symbolic link.
// Analogous to: [os.Readlink], readlink, 9P2000.u Treadlink.
// If the link destination is relative, ReadLink returns the relative path
// without resolving it to an absolute one.
//
// Requires: [ReadLinkFS]
func ReadLink(ctx context.Context, fsys FS, name string) (string, error) {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return "", err
	}
	if rfs, ok := fsys.(ReadLinkFS); ok {
		return rfs.ReadLink(ctx, name)
	}
	return "", &PathError{
		Op:   "readlink",
		Path: name,
		Err:  ErrUnsupported,
	}
}

// Lstat returns FileInfo describing the named file.
// Analogous to: [os.Lstat], stat -L, ls -l (on symlink itself).
// If the file is a symbolic link, the returned FileInfo describes the
// symbolic link. Lstat makes no attempt to follow the link.
//
// Requires: [ReadLinkFS] || [StatFS]
func Lstat(ctx context.Context, fsys FS, name string) (FileInfo, error) {
	var err error
	if name, err = localizePath(ctx, fsys, name); err != nil {
		return nil, err
	}
	if rfs, ok := fsys.(ReadLinkFS); ok {
		return rfs.Lstat(ctx, name)
	}
	return Stat(ctx, fsys, name)
}
