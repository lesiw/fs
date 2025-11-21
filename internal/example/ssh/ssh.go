// Package sshfs provides an SSH/SFTP filesystem implementation.
//
// This is a sketch/example implementation to demonstrate how lesiw.io/fs
// can be used with remote filesystems over SSH/SFTP.
//
// This implementation is NOT production-ready and should not be used
// outside of examples and testing.
package ssh

import (
	"context"
	"errors"
	"io"
	"iter"
	"os"
	"path"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"lesiw.io/fs"
)

// SSHFS implements lesiw.io/fs.FS using SFTP over SSH.
type SSHFS struct {
	client *sftp.Client
	conn   *ssh.Client
	prefix string
}

// New creates a new SSHFS instance connected to the given SSH server.
func New(addr, user, password string) (*SSHFS, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &SSHFS{
		client: client,
		conn:   conn,
		prefix: "",
	}, nil
}

// SetPrefix sets a path prefix for all operations.
// This is useful when the SFTP server restricts access to a subdirectory.
func (f *SSHFS) SetPrefix(prefix string) {
	f.prefix = prefix
}

func (f *SSHFS) fullPath(ctx context.Context, name string) string {
	if workDir := fs.WorkDir(ctx); workDir != "" {
		name = path.Join(workDir, name)
	}
	if f.prefix != "" {
		name = path.Join(f.prefix, name)
	}
	return name
}

// Close closes the SFTP and SSH connections.
func (f *SSHFS) Close() error {
	if err := f.client.Close(); err != nil {
		_ = f.conn.Close()
		return err
	}
	return f.conn.Close()
}

// Open implements fs.FS.
func (f *SSHFS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.client.Open(f.fullPath(ctx, name))
	if err != nil {
		return nil, convertError("open", name, err)
	}

	return file, nil
}

// Create implements fs.CreateFS.
func (f *SSHFS) Create(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "create",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.client.OpenFile(
		f.fullPath(ctx, name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
	)
	if err != nil {
		return nil, convertError("create", name, err)
	}

	if err := file.Chmod(os.FileMode(fs.FileMode(ctx))); err != nil {
		_ = file.Close()
		return nil, convertError("chmod", name, err)
	}

	return file, nil
}

// Append implements fs.AppendFS.
func (f *SSHFS) Append(
	ctx context.Context, name string,
) (io.WriteCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "append",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.client.OpenFile(
		f.fullPath(ctx, name), os.O_WRONLY|os.O_CREATE|os.O_APPEND,
	)
	if err != nil {
		return nil, convertError("append", name, err)
	}

	if err := file.Chmod(os.FileMode(fs.FileMode(ctx))); err != nil {
		_ = file.Close()
		return nil, convertError("chmod", name, err)
	}

	return file, nil
}

// Stat implements fs.StatFS.
func (f *SSHFS) Stat(
	ctx context.Context, name string,
) (fs.FileInfo, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	info, err := f.client.Stat(f.fullPath(ctx, name))
	if err != nil {
		return nil, convertError("stat", name, err)
	}

	return &sshFileInfo{info: info}, nil
}

// ReadDir implements fs.ReadDirFS.
func (f *SSHFS) ReadDir(
	ctx context.Context, name string,
) iter.Seq2[fs.DirEntry, error] {
	return func(yield func(fs.DirEntry, error) bool) {
		if name == "" {
			name = "."
		}

		entries, err := f.client.ReadDir(f.fullPath(ctx, name))
		if err != nil {
			yield(nil, convertError("readdir", name, err))
			return
		}

		for _, entry := range entries {
			if !yield(&sshDirEntry{info: entry}, nil) {
				return
			}
		}
	}
}

// Mkdir implements fs.MkdirFS.
func (f *SSHFS) Mkdir(
	ctx context.Context, name string,
) error {
	if name == "" {
		return &fs.PathError{
			Op:   "mkdir",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.client.Mkdir(f.fullPath(ctx, name))
	if err != nil {
		return convertError("mkdir", name, err)
	}

	mode := os.FileMode(fs.DirMode(ctx))
	if err := f.client.Chmod(f.fullPath(ctx, name), mode); err != nil {
		return convertError("chmod", name, err)
	}

	return nil
}

// Remove implements fs.RemoveFS.
func (f *SSHFS) Remove(ctx context.Context, name string) error {
	if name == "" {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.client.Remove(f.fullPath(ctx, name))
	if err != nil {
		return convertError("remove", name, err)
	}

	return nil
}

// Rename implements fs.RenameFS.
func (f *SSHFS) Rename(
	ctx context.Context, oldname, newname string,
) error {
	if oldname == "" || newname == "" {
		return &fs.PathError{
			Op:   "rename",
			Path: oldname,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.client.Rename(f.fullPath(ctx, oldname), f.fullPath(ctx, newname))
	if err != nil {
		return convertError("rename", oldname, err)
	}

	return nil
}

// Chmod implements fs.ChmodFS.
func (f *SSHFS) Chmod(
	ctx context.Context, name string, mode fs.Mode,
) error {
	if name == "" {
		return &fs.PathError{
			Op:   "chmod",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.client.Chmod(f.fullPath(ctx, name), os.FileMode(mode))
	if err != nil {
		return convertError("chmod", name, err)
	}

	return nil
}

// Chown implements fs.ChownFS.
func (f *SSHFS) Chown(ctx context.Context, name string, uid, gid int) error {
	if name == "" {
		return &fs.PathError{
			Op:   "chown",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.client.Chown(f.fullPath(ctx, name), uid, gid)
	if err != nil {
		return convertError("chown", name, err)
	}

	return nil
}

// Chtimes implements fs.ChtimesFS.
func (f *SSHFS) Chtimes(
	ctx context.Context, name string, atime, mtime time.Time,
) error {
	if name == "" {
		return &fs.PathError{
			Op:   "chtimes",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.client.Chtimes(f.fullPath(ctx, name), atime, mtime)
	if err != nil {
		return convertError("chtimes", name, err)
	}

	return nil
}

// Symlink implements fs.SymlinkFS.
func (f *SSHFS) Symlink(
	ctx context.Context, oldname, newname string,
) error {
	if oldname == "" || newname == "" {
		return &fs.PathError{
			Op:   "symlink",
			Path: newname,
			Err:  fs.ErrInvalid,
		}
	}

	err := f.client.Symlink(oldname, f.fullPath(ctx, newname))
	if err != nil {
		return convertError("symlink", newname, err)
	}

	return nil
}

// ReadLink implements fs.ReadLinkFS.
func (f *SSHFS) ReadLink(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", &fs.PathError{
			Op:   "readlink",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	target, err := f.client.ReadLink(f.fullPath(ctx, name))
	if err != nil {
		return "", convertError("readlink", name, err)
	}

	return target, nil
}

// convertError converts SFTP errors to lesiw.io/fs errors.
func convertError(op, path string, err error) error {
	if err == nil {
		return nil
	}

	var fsErr error
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		// Extract the underlying error from os.PathError
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

// sshFileInfo wraps os.FileInfo to implement fs.FileInfo.
type sshFileInfo struct {
	info os.FileInfo
}

func (fi *sshFileInfo) Name() string       { return fi.info.Name() }
func (fi *sshFileInfo) Size() int64        { return fi.info.Size() }
func (fi *sshFileInfo) ModTime() time.Time { return fi.info.ModTime() }
func (fi *sshFileInfo) IsDir() bool        { return fi.info.IsDir() }
func (fi *sshFileInfo) Sys() any           { return fi.info.Sys() }
func (fi *sshFileInfo) Mode() fs.Mode      { return fs.Mode(fi.info.Mode()) }

// sshDirEntry wraps os.FileInfo to implement fs.DirEntry.
type sshDirEntry struct {
	info os.FileInfo
}

func (de *sshDirEntry) Name() string { return de.info.Name() }
func (de *sshDirEntry) IsDir() bool  { return de.info.IsDir() }
func (de *sshDirEntry) Type() fs.Mode {
	return fs.Mode(de.info.Mode().Type())
}

func (de *sshDirEntry) Info() (fs.FileInfo, error) {
	return &sshFileInfo{info: de.info}, nil
}

func (de *sshDirEntry) Path() string { return "" }
