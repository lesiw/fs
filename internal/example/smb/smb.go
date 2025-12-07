// Package smb provides a Samba/SMB/CIFS filesystem implementation.
//
// This is a sketch/example implementation to demonstrate how lesiw.io/fs
// can be used with SMB (Server Message Block) file shares.
//
// This implementation is NOT production-ready and should not be used
// outside of examples and testing.
package smb

import (
	"context"
	"errors"
	"io"
	"iter"
	"net"
	"os"
	"path"
	"time"

	"github.com/hirochachacha/go-smb2"

	"lesiw.io/fs"
)

// FS implements lesiw.io/fs.FS using SMB/CIFS.
type smbFS struct {
	session *smb2.Session
	share   *smb2.Share
}

// New creates a new SMB filesystem client.
//
// addr: SMB server address (e.g., "localhost:445")
// shareName: Share name to connect to (e.g., "public")
// user: Username for authentication
// password: Password for authentication
func New(addr, shareName, user, password string) (fs.FS, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     user,
			Password: password,
		},
	}

	session, err := d.Dial(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	share, err := session.Mount(shareName)
	if err != nil {
		_ = session.Logoff()
		return nil, err
	}

	return &smbFS{
		session: session,
		share:   share,
	}, nil
}

// Close closes the SMB share and session.
func (f *smbFS) Close() error {
	if err := f.share.Umount(); err != nil {
		_ = f.session.Logoff()
		return err
	}
	return f.session.Logoff()
}

func (f *smbFS) fullPath(ctx context.Context, name string) string {
	if workDir := fs.WorkDir(ctx); workDir != "" {
		name = path.Join(workDir, name)
	}
	return name
}

// Open implements fs.FS.
func (f *smbFS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.share.Open(f.fullPath(ctx, name))
	if err != nil {
		return nil, convertError("open", name, err)
	}

	return file, nil
}

// Create implements fs.CreateFS.
func (f *smbFS) Create(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "create",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.share.OpenFile(
		f.fullPath(ctx, name),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		os.FileMode(fs.FileMode(ctx)),
	)
	if err != nil {
		return nil, convertError("create", name, err)
	}

	return file, nil
}

// Append implements fs.AppendFS.
func (f *smbFS) Append(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "append",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.share.OpenFile(
		f.fullPath(ctx, name),
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		os.FileMode(fs.FileMode(ctx)),
	)
	if err != nil {
		return nil, convertError("append", name, err)
	}

	return file, nil
}

// Stat implements fs.StatFS.
func (f *smbFS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	info, err := f.share.Stat(f.fullPath(ctx, name))
	if err != nil {
		return nil, convertError("stat", name, err)
	}

	return &fileInfo{info: info}, nil
}

// ReadDir implements fs.ReadDirFS.
func (f *smbFS) ReadDir(
	ctx context.Context, name string,
) iter.Seq2[fs.DirEntry, error] {
	return func(yield func(fs.DirEntry, error) bool) {
		if name == "" {
			name = "."
		}

		fullPath := f.fullPath(ctx, name)
		entries, err := f.share.ReadDir(fullPath)
		if err != nil {
			yield(nil, convertError("readdir", name, err))
			return
		}

		for _, entry := range entries {
			// Skip .deleted directory used by SMB for soft deletes
			if entry.Name() == ".deleted" {
				continue
			}
			if !yield(&dirEntry{info: entry}, nil) {
				return
			}
		}
	}
}

// Mkdir implements fs.MkdirFS.
func (f *smbFS) Mkdir(
	ctx context.Context, name string,
) error {
	if name == "" {
		return &fs.PathError{
			Op:   "mkdir",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.share.Mkdir(f.fullPath(ctx, name), os.FileMode(fs.DirMode(ctx)))
	if err != nil {
		return convertError("mkdir", name, err)
	}

	return nil
}

// Remove implements fs.RemoveFS.
func (f *smbFS) Remove(ctx context.Context, name string) error {
	if name == "" {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	fullPath := f.fullPath(ctx, name)
	if err := f.share.Remove(fullPath); err != nil {
		return convertError("remove", name, err)
	}

	return nil
}

// RemoveAll implements fs.RemoveAllFS to work around go-smb2 bugs where
// Stat() and Remove() hang on directories in certain states.
func (f *smbFS) RemoveAll(ctx context.Context, name string) error {
	if name == "" {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	fullPath := f.fullPath(ctx, name)

	// Try to read as directory first - if it fails, try to remove as file
	for entry, readErr := range f.ReadDir(ctx, name) {
		if readErr != nil {
			// Not a directory or doesn't exist - try to remove as file
			if err := f.share.Remove(fullPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return nil // Already gone
				}
				return convertError("remove", name, err)
			}
			return nil
		}
		childName := path.Join(name, entry.Name())
		if err := f.RemoveAll(ctx, childName); err != nil {
			return err
		}
	}

	// Remove the now-empty directory (or empty directory from the start)
	if err := f.share.Remove(fullPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // Already gone
		}
		return convertError("remove", name, err)
	}

	return nil
}

// Rename implements fs.RenameFS.
func (f *smbFS) Rename(
	ctx context.Context, oldname, newname string,
) error {
	if oldname == "" || newname == "" {
		return &fs.PathError{
			Op:   "rename",
			Path: oldname,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.share.Rename(f.fullPath(ctx, oldname), f.fullPath(ctx, newname))
	if err != nil {
		return convertError("rename", oldname, err)
	}

	return nil
}

// convertError converts SMB/OS errors to lesiw.io/fs errors.
func convertError(op, path string, err error) error {
	if err == nil {
		return nil
	}

	var fsErr error
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}

	switch {
	case errors.Is(err, os.ErrNotExist):
		fsErr = fs.ErrNotExist
	case errors.Is(err, os.ErrExist):
		fsErr = fs.ErrExist
	case errors.Is(err, os.ErrPermission):
		fsErr = fs.ErrPermission
	case errors.Is(err, os.ErrInvalid):
		fsErr = fs.ErrInvalid
	default:
		fsErr = err
	}

	return &fs.PathError{
		Op:   op,
		Path: path,
		Err:  fsErr,
	}
}

// fileInfo wraps os.FileInfo to implement fs.FileInfo.
type fileInfo struct {
	info os.FileInfo
}

func (fi *fileInfo) Name() string       { return fi.info.Name() }
func (fi *fileInfo) Size() int64        { return fi.info.Size() }
func (fi *fileInfo) ModTime() time.Time { return fi.info.ModTime() }
func (fi *fileInfo) IsDir() bool        { return fi.info.IsDir() }
func (fi *fileInfo) Sys() any           { return fi.info.Sys() }
func (fi *fileInfo) Mode() fs.Mode      { return fs.Mode(fi.info.Mode()) }

// dirEntry wraps os.FileInfo to implement fs.DirEntry.
type dirEntry struct {
	info os.FileInfo
}

func (de *dirEntry) Name() string { return de.info.Name() }
func (de *dirEntry) IsDir() bool  { return de.info.IsDir() }
func (de *dirEntry) Type() fs.Mode {
	return fs.Mode(de.info.Mode().Type())
}

func (de *dirEntry) Info() (fs.FileInfo, error) {
	return &fileInfo{info: de.info}, nil
}

func (de *dirEntry) Path() string { return "" }

// Abs implements fs.AbsFS
func (f *smbFS) Abs(ctx context.Context, name string) (string, error) {
	// If already absolute, return as-is
	if path.IsAbs(name) {
		return path.Clean(name), nil
	}

	// If we have an absolute WorkDir, we can resolve the path
	if workDir := fs.WorkDir(ctx); workDir != "" && path.IsAbs(workDir) {
		return path.Join(workDir, name), nil
	}

	// Otherwise, we can't determine an absolute path
	return "", &fs.PathError{Op: "abs", Path: name, Err: fs.ErrUnsupported}
}
