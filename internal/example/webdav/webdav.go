// Package webdav provides a WebDAV filesystem implementation.
//
// This is a sketch/example implementation to demonstrate how lesiw.io/fs
// can be used with WebDAV servers.
//
// This implementation is NOT production-ready and should not be used outside
// of examples and testing.
package webdav

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"path"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"

	"lesiw.io/fs"
)

// FS implements fs.FS for WebDAV servers.
type FS struct {
	client *gowebdav.Client
}

// New creates a new WebDAV filesystem.
//
// url: WebDAV server URL (e.g., "http://localhost:8080/webdav")
// user: Username for authentication
// password: Password for authentication
func New(url, user, password string) (*FS, error) {
	client := gowebdav.NewClient(url, user, password)

	// Test connection
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("connecting to WebDAV server: %w", err)
	}

	return &FS{client: client}, nil
}

// fullPath resolves the full path by prepending the working directory from
// context if present.
func (f *FS) fullPath(ctx context.Context, name string) string {
	if workDir := fs.WorkDir(ctx); workDir != "" {
		name = path.Join(workDir, name)
	}
	return name
}

// Open implements fs.FS
func (f *FS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	data, err := f.client.Read(f.fullPath(ctx, name))
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  err,
		}
	}

	return &webdavReadCloser{Reader: bytes.NewReader(data)}, nil
}

// Create implements fs.CreateFS
func (f *FS) Create(ctx context.Context, name string) (io.WriteCloser, error) {
	return &webdavWriteCloser{
		client:     f.client,
		name:       f.fullPath(ctx, name),
		buf:        &bytes.Buffer{},
		mustUpload: true,
	}, nil
}

// Append implements fs.AppendFS
func (f *FS) Append(ctx context.Context, name string) (io.WriteCloser, error) {
	fullPath := f.fullPath(ctx, name)
	wc := &webdavWriteCloser{
		client:     f.client,
		name:       fullPath,
		buf:        &bytes.Buffer{},
		mustUpload: true,
	}

	data, err := f.client.Read(fullPath)
	if err == nil {
		wc.buf.Write(data)
	}

	return wc, nil
}

// webdavReadCloser wraps a byte reader
type webdavReadCloser struct {
	*bytes.Reader
}

func (r *webdavReadCloser) Close() error {
	return nil
}

// webdavWriteCloser buffers writes and uploads on Close
type webdavWriteCloser struct {
	client     *gowebdav.Client
	name       string
	buf        *bytes.Buffer
	mustUpload bool
}

func (w *webdavWriteCloser) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *webdavWriteCloser) Close() error {
	if w.buf == nil || w.buf.Len() == 0 {
		// Check if we must upload (O_TRUNC case)
		if !w.mustUpload {
			// No writes and no truncate, just close
			return nil
		}
		// Must upload empty file for truncate
		w.buf = &bytes.Buffer{}
	}
	return w.client.Write(w.name, w.buf.Bytes(), 0644)
}

// Stat implements fs.StatFS
func (f *FS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := f.client.Stat(f.fullPath(ctx, name))
	if err != nil {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  err,
		}
	}

	return &webdavFileInfo{
		name: path.Base(name),
		size: info.Size(),
		mode: info.Mode(),
		time: info.ModTime(),
	}, nil
}

// ReadDir implements fs.ReadDirFS
func (f *FS) ReadDir(
	ctx context.Context, name string,
) iter.Seq2[fs.DirEntry, error] {
	return func(yield func(fs.DirEntry, error) bool) {
		fullPath := f.fullPath(ctx, name)
		if fullPath == "." {
			fullPath = "/"
		} else if !strings.HasPrefix(fullPath, "/") {
			fullPath = "/" + fullPath
		}

		infos, err := f.client.ReadDir(fullPath)
		if err != nil {
			yield(nil, &fs.PathError{
				Op:   "readdir",
				Path: fullPath,
				Err:  err,
			})
			return
		}

		for _, info := range infos {
			entry := &webdavDirEntry{
				name:  info.Name(),
				isDir: info.IsDir(),
				size:  info.Size(),
				mode:  info.Mode(),
				time:  info.ModTime(),
			}
			if !yield(entry, nil) {
				return
			}
		}
	}
}

// Remove implements fs.RemoveFS
func (f *FS) Remove(ctx context.Context, name string) error {
	fullPath := f.fullPath(ctx, name)
	// Check if this is a directory
	info, statErr := f.Stat(ctx, name)
	if statErr == nil && info.IsDir() {
		// Check if directory has children
		for _, readErr := range f.ReadDir(ctx, name) {
			if readErr != nil {
				break
			}
			// Found at least one entry - directory not empty
			return &fs.PathError{
				Op:   "remove",
				Path: fullPath,
				Err:  fmt.Errorf("directory not empty"),
			}
		}
	}

	err := f.client.Remove(fullPath)
	if err != nil {
		return &fs.PathError{
			Op:   "remove",
			Path: fullPath,
			Err:  err,
		}
	}
	return nil
}

// Mkdir implements fs.MkdirFS
func (f *FS) Mkdir(ctx context.Context, name string) error {
	perm := fs.DirMode(ctx)
	err := f.client.Mkdir(f.fullPath(ctx, name), perm)
	if err != nil {
		return &fs.PathError{
			Op:   "mkdir",
			Path: name,
			Err:  err,
		}
	}
	return nil
}

// Rename implements fs.RenameFS
func (f *FS) Rename(ctx context.Context, oldname, newname string) error {
	err := f.client.Rename(
		f.fullPath(ctx, oldname), f.fullPath(ctx, newname), false,
	)
	if err != nil {
		return &fs.PathError{
			Op:   "rename",
			Path: oldname,
			Err:  err,
		}
	}
	return nil
}

// webdavFileInfo implements fs.FileInfo
type webdavFileInfo struct {
	name string
	size int64
	mode fs.Mode
	time time.Time
}

func (fi *webdavFileInfo) Name() string       { return fi.name }
func (fi *webdavFileInfo) Size() int64        { return fi.size }
func (fi *webdavFileInfo) Mode() fs.Mode      { return fi.mode }
func (fi *webdavFileInfo) ModTime() time.Time { return fi.time }
func (fi *webdavFileInfo) Sys() any           { return nil }

func (fi *webdavFileInfo) IsDir() bool { return fi.mode.IsDir() }

// webdavDirEntry implements fs.DirEntry
type webdavDirEntry struct {
	name  string
	isDir bool
	size  int64
	mode  fs.Mode
	time  time.Time
}

func (de *webdavDirEntry) Name() string  { return de.name }
func (de *webdavDirEntry) IsDir() bool   { return de.isDir }
func (de *webdavDirEntry) Type() fs.Mode { return de.mode.Type() }
func (de *webdavDirEntry) Path() string  { return "" }

func (de *webdavDirEntry) Info() (fs.FileInfo, error) {
	return &webdavFileInfo{
		name: de.name,
		size: de.size,
		mode: de.mode,
		time: de.time,
	}, nil
}

// Abs implements fs.AbsFS
func (f *FS) Abs(ctx context.Context, name string) (string, error) {
	// WebDAV URLs can be absolute, return as-is if already absolute
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

// Compile-time interface checks
var (
	_ fs.FS        = (*FS)(nil)
	_ fs.CreateFS  = (*FS)(nil)
	_ fs.AppendFS  = (*FS)(nil)
	_ fs.StatFS    = (*FS)(nil)
	_ fs.ReadDirFS = (*FS)(nil)
	_ fs.RemoveFS  = (*FS)(nil)
	_ fs.MkdirFS   = (*FS)(nil)
	_ fs.RenameFS  = (*FS)(nil)
	_ fs.AbsFS     = (*FS)(nil)
)
