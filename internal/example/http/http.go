// Package httpfs provides an HTTP-based filesystem implementation.
//
// This is a sketch/example implementation to demonstrate how lesiw.io/fs
// can be used with HTTP servers that provide directory listings.
//
// This implementation is read-only and NOT production-ready. It should
// not be used outside of examples and testing.
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"lesiw.io/fs"
)

// httpFS implements a read-only lesiw.io/fs.FS using HTTP.
type httpFS struct {
	baseURL string
	client  *http.Client
}

// New creates a new HTTP filesystem for the given base URL.
func New(baseURL string) fs.FS {
	return &httpFS{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Open implements fs.FS (read-only).
func (f *httpFS) Open(
	ctx context.Context, name string,
) (io.ReadCloser, error) {
	if name == "" {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	url := f.baseURL + "/" + name
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, convertError("open", name, err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return nil, &fs.PathError{
				Op:   "open",
				Path: name,
				Err:  fs.ErrNotExist,
			}
		}
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status),
		}
	}

	return resp.Body, nil
}

// Stat implements fs.StatFS.
func (f *httpFS) Stat(
	ctx context.Context, name string,
) (fs.FileInfo, error) {
	if name == "" || name == "." {
		return &httpFileInfo{
			name:  ".",
			isDir: true,
			size:  0,
			time:  time.Now(),
		}, nil
	}

	url := f.baseURL + "/" + name
	resp, err := f.client.Head(url)
	if err != nil {
		return nil, convertError("stat", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status),
		}
	}

	size := resp.ContentLength
	modTime := time.Now()
	if lastMod := resp.Header.Get("Last-Modified"); lastMod != "" {
		if t, err := http.ParseTime(lastMod); err == nil {
			modTime = t
		}
	}

	// Detect if this is a directory
	// HTTP file servers typically serve directories with text/html content
	// type (directory listing), while files have their actual content type
	isDir := false
	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/html") {
		isDir = true
	}

	return &httpFileInfo{
		name:  path.Base(name),
		isDir: isDir,
		size:  size,
		time:  modTime,
	}, nil
}

// convertError converts HTTP errors to lesiw.io/fs errors.
func convertError(op, path string, err error) error {
	if err == nil {
		return nil
	}

	return &fs.PathError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}

// httpFileInfo implements fs.FileInfo for HTTP resources.
type httpFileInfo struct {
	name  string
	isDir bool
	size  int64
	time  time.Time
}

func (fi *httpFileInfo) Name() string       { return fi.name }
func (fi *httpFileInfo) Size() int64        { return fi.size }
func (fi *httpFileInfo) ModTime() time.Time { return fi.time }
func (fi *httpFileInfo) IsDir() bool        { return fi.isDir }
func (fi *httpFileInfo) Sys() any           { return nil }

func (fi *httpFileInfo) Mode() fs.Mode {
	if fi.isDir {
		return fs.Mode(0555 | fs.ModeDir)
	}
	return fs.Mode(0444)
}

// Abs implements fs.AbsFS
func (f *httpFS) Abs(ctx context.Context, name string) (string, error) {
	// Resolve with WorkDir if present
	fullPath := name
	if workDir := fs.WorkDir(ctx); workDir != "" {
		fullPath = path.Join(workDir, name)
	}

	// Clean the path
	cleanPath := path.Clean(fullPath)

	// Join with base URL
	if path.IsAbs(cleanPath) {
		return f.baseURL + cleanPath, nil
	}

	// Relative path - prepend /
	return f.baseURL + "/" + cleanPath, nil
}
