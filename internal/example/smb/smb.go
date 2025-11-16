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
	"time"

	"github.com/hirochachacha/go-smb2"

	"lesiw.io/fs"
)

// FS implements lesiw.io/fs.FS using SMB/CIFS.
type FS struct {
	session *smb2.Session
	share   *smb2.Share
}

// New creates a new SMB filesystem client.
//
// addr: SMB server address (e.g., "localhost:445")
// shareName: Share name to connect to (e.g., "public")
// user: Username for authentication
// password: Password for authentication
func New(addr, shareName, user, password string) (*FS, error) {
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

	return &FS{
		session: session,
		share:   share,
	}, nil
}

// Close closes the SMB share and session.
func (f *FS) Close() error {
	if err := f.share.Umount(); err != nil {
		_ = f.session.Logoff()
		return err
	}
	return f.session.Logoff()
}

// Open implements fs.FS.
func (f *FS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.share.Open(name)
	if err != nil {
		return nil, convertError("open", name, err)
	}

	return file, nil
}

// Create implements fs.CreateFS.
func (f *FS) Create(ctx context.Context, name string) (io.WriteCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "create",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.share.OpenFile(
		name,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		os.FileMode(fs.FileMode(ctx)),
	)
	if err != nil {
		return nil, convertError("create", name, err)
	}

	return file, nil
}

// Append implements fs.AppendFS.
func (f *FS) Append(ctx context.Context, name string) (io.WriteCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "append",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.share.OpenFile(
		name,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		os.FileMode(fs.FileMode(ctx)),
	)
	if err != nil {
		return nil, convertError("append", name, err)
	}

	return file, nil
}

// Stat implements fs.StatFS.
func (f *FS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	info, err := f.share.Stat(name)
	if err != nil {
		return nil, convertError("stat", name, err)
	}

	return &fileInfo{info: info}, nil
}

// ReadDir implements fs.ReadDirFS.
func (f *FS) ReadDir(
	ctx context.Context, name string,
) iter.Seq2[fs.DirEntry, error] {
	return func(yield func(fs.DirEntry, error) bool) {
		if name == "" {
			name = "."
		}

		entries, err := f.share.ReadDir(name)
		if err != nil {
			yield(nil, convertError("readdir", name, err))
			return
		}

		for _, entry := range entries {
			if !yield(&dirEntry{info: entry}, nil) {
				return
			}
		}
	}
}

// Mkdir implements fs.MkdirFS.
func (f *FS) Mkdir(
	ctx context.Context, name string,
) error {
	if name == "" {
		return &fs.PathError{
			Op:   "mkdir",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	if err := f.share.Mkdir(name, os.FileMode(fs.DirMode(ctx))); err != nil {
		return convertError("mkdir", name, err)
	}

	return nil
}

// Remove implements fs.RemoveFS.
func (f *FS) Remove(ctx context.Context, name string) error {
	if name == "" {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	if err := f.share.Remove(name); err != nil {
		return convertError("remove", name, err)
	}

	return nil
}

// Rename implements fs.RenameFS.
func (f *FS) Rename(
	ctx context.Context, oldname, newname string,
) error {
	if oldname == "" || newname == "" {
		return &fs.PathError{
			Op:   "rename",
			Path: oldname,
			Err:  fs.ErrInvalid,
		}
	}

	if err := f.share.Rename(oldname, newname); err != nil {
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
