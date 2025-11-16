// Package sftp provides a dedicated SFTP filesystem implementation.
//
// This is a sketch/example implementation to demonstrate how lesiw.io/fs
// can be used with SFTP servers (SSH File Transfer Protocol).
//
// Unlike the ssh package which establishes a full SSH connection,
// this implementation focuses solely on SFTP file transfer operations.
//
// This implementation is NOT production-ready and should not be used
// outside of examples and testing.
package sftp

import (
	"context"
	"errors"
	"io"
	"iter"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"lesiw.io/fs"
)

// FS implements lesiw.io/fs.FS using SFTP.
type FS struct {
	client   *sftp.Client
	sshConn  *ssh.Client
	basePath string
}

// New creates a new SFTP filesystem client.
//
// addr: SFTP server address (e.g., "localhost:22")
// user: Username for authentication
// password: Password for authentication
func New(addr, user, password string) (*FS, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// Establish SSH connection (required for SFTP)
	sshConn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	// Create SFTP client over SSH connection
	client, err := sftp.NewClient(sshConn)
	if err != nil {
		_ = sshConn.Close()
		return nil, err
	}

	return &FS{
		client:   client,
		sshConn:  sshConn,
		basePath: "",
	}, nil
}

// SetBasePath sets a base path prefix for all operations.
// Useful when the SFTP server restricts access to a subdirectory.
func (f *FS) SetBasePath(path string) {
	f.basePath = path
}

func (f *FS) fullPath(name string) string {
	if f.basePath == "" {
		return name
	}
	if name == "" || name == "." {
		return f.basePath
	}
	return f.basePath + "/" + name
}

// Close closes the SFTP client and underlying SSH connection.
func (f *FS) Close() error {
	if err := f.client.Close(); err != nil {
		_ = f.sshConn.Close()
		return err
	}
	return f.sshConn.Close()
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

	file, err := f.client.Open(f.fullPath(name))
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

	file, err := f.client.OpenFile(
		f.fullPath(name), os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
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
func (f *FS) Append(ctx context.Context, name string) (io.WriteCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "append",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	file, err := f.client.OpenFile(
		f.fullPath(name), os.O_WRONLY|os.O_CREATE|os.O_APPEND,
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
func (f *FS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	info, err := f.client.Stat(f.fullPath(name))
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

		entries, err := f.client.ReadDir(f.fullPath(name))
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

	if err := f.client.Mkdir(f.fullPath(name)); err != nil {
		return convertError("mkdir", name, err)
	}

	mode := os.FileMode(fs.DirMode(ctx))
	if err := f.client.Chmod(f.fullPath(name), mode); err != nil {
		return convertError("chmod", name, err)
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

	if err := f.client.Remove(f.fullPath(name)); err != nil {
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

	err := f.client.Rename(f.fullPath(oldname), f.fullPath(newname))
	if err != nil {
		return convertError("rename", oldname, err)
	}

	return nil
}

// Chmod implements fs.ChmodFS.
func (f *FS) Chmod(
	ctx context.Context, name string, mode fs.Mode,
) error {
	if name == "" {
		return &fs.PathError{
			Op:   "chmod",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	if err := f.client.Chmod(f.fullPath(name), os.FileMode(mode)); err != nil {
		return convertError("chmod", name, err)
	}

	return nil
}

// Chown implements fs.ChownFS.
func (f *FS) Chown(ctx context.Context, name string, uid, gid int) error {
	if name == "" {
		return &fs.PathError{
			Op:   "chown",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	if err := f.client.Chown(f.fullPath(name), uid, gid); err != nil {
		return convertError("chown", name, err)
	}

	return nil
}

// Chtimes implements fs.ChtimesFS.
func (f *FS) Chtimes(
	ctx context.Context, name string, atime, mtime time.Time,
) error {
	if name == "" {
		return &fs.PathError{
			Op:   "chtimes",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	if err := f.client.Chtimes(f.fullPath(name), atime, mtime); err != nil {
		return convertError("chtimes", name, err)
	}

	return nil
}

// Symlink implements fs.SymlinkFS.
func (f *FS) Symlink(
	ctx context.Context, oldname, newname string,
) error {
	if oldname == "" || newname == "" {
		return &fs.PathError{
			Op:   "symlink",
			Path: newname,
			Err:  fs.ErrInvalid,
		}
	}

	if err := f.client.Symlink(oldname, f.fullPath(newname)); err != nil {
		return convertError("symlink", newname, err)
	}

	return nil
}

// ReadLink implements fs.ReadLinkFS.
func (f *FS) ReadLink(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", &fs.PathError{
			Op:   "readlink",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	target, err := f.client.ReadLink(f.fullPath(name))
	if err != nil {
		return "", convertError("readlink", name, err)
	}

	return target, nil
}

// convertError converts SFTP/OS errors to lesiw.io/fs errors.
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
