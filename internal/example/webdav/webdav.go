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

// Open implements fs.FS
func (f *FS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	data, err := f.client.Read(name)
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
		name:       name,
		buf:        &bytes.Buffer{},
		mustUpload: true,
	}, nil
}

// Append implements fs.AppendFS
func (f *FS) Append(ctx context.Context, name string) (io.WriteCloser, error) {
	wc := &webdavWriteCloser{
		client:     f.client,
		name:       name,
		buf:        &bytes.Buffer{},
		mustUpload: true,
	}

	data, err := f.client.Read(name)
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
	info, err := f.client.Stat(name)
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
		if name == "." {
			name = "/"
		} else if !strings.HasPrefix(name, "/") {
			name = "/" + name
		}

		infos, err := f.client.ReadDir(name)
		if err != nil {
			yield(nil, &fs.PathError{
				Op:   "readdir",
				Path: name,
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
				Path: name,
				Err:  fmt.Errorf("directory not empty"),
			}
		}
	}

	err := f.client.Remove(name)
	if err != nil {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  err,
		}
	}
	return nil
}

// Mkdir implements fs.MkdirFS
func (f *FS) Mkdir(ctx context.Context, name string) error {
	perm := fs.DirMode(ctx)
	err := f.client.Mkdir(name, perm)
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
	err := f.client.Rename(oldname, newname, false)
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
)
