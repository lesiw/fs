// Package s3 provides an S3-compatible filesystem implementation.
//
// This is a sketch/example implementation to demonstrate how lesiw.io/fs
// can be used with cloud storage backends like S3, MinIO, or other
// S3-compatible services.
//
// This implementation is NOT production-ready and should not be used outside
// of examples and testing.
package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"lesiw.io/fs"
)

// FS implements fs.FS for S3-compatible object storage.
type FS struct {
	client *minio.Client
	bucket string
}

// New creates a new S3 filesystem.
//
// endpoint: S3 endpoint (e.g., "localhost:9000" for MinIO)
// bucket: S3 bucket name
// accessKey: S3 access key
// secretKey: S3 secret key
// useSSL: whether to use HTTPS
func New(
	endpoint, bucket, accessKey, secretKey string, useSSL bool,
) (*FS, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating minio client: %w", err)
	}

	return &FS{
		client: client,
		bucket: bucket,
	}, nil
}

var _ fs.FS = (*FS)(nil)

func (f *FS) Open(ctx context.Context, name string) (io.ReadCloser, error) {
	obj, err := f.client.GetObject(
		ctx, f.bucket, name, minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  err,
		}
	}

	return obj, nil
}

var _ fs.CreateFS = (*FS)(nil)

func (f *FS) Create(ctx context.Context, name string) (io.WriteCloser, error) {
	return &s3WriteCloser{
		ctx:        ctx,
		client:     f.client,
		bucket:     f.bucket,
		name:       name,
		mustUpload: true,
	}, nil
}

var _ fs.AppendFS = (*FS)(nil)

func (f *FS) Append(ctx context.Context, name string) (io.WriteCloser, error) {
	wc := &s3WriteCloser{
		ctx:        ctx,
		client:     f.client,
		bucket:     f.bucket,
		name:       name,
		mustUpload: true,
	}

	obj, err := f.client.GetObject(
		ctx, f.bucket, name, minio.GetObjectOptions{},
	)
	if err == nil {
		wc.buf = &bytes.Buffer{}
		_, readErr := io.Copy(wc.buf, obj)
		_ = obj.Close()
		if readErr != nil {
			return nil, &fs.PathError{
				Op:   "append",
				Path: name,
				Err:  readErr,
			}
		}
	}

	return wc, nil
}

// s3WriteCloser buffers writes and uploads on Close
type s3WriteCloser struct {
	ctx        context.Context
	client     *minio.Client
	bucket     string
	name       string
	buf        *bytes.Buffer
	mustUpload bool
}

func (w *s3WriteCloser) Write(p []byte) (int, error) {
	if w.buf == nil {
		w.buf = &bytes.Buffer{}
	}
	return w.buf.Write(p)
}

func (w *s3WriteCloser) Close() error {
	if w.buf == nil || w.buf.Len() == 0 {
		// Check if we must upload (O_TRUNC case)
		if !w.mustUpload {
			// No writes and no truncate, just close
			return nil
		}
		// Must upload empty file for truncate
		w.buf = &bytes.Buffer{}
	}

	// Upload buffered content
	_, err := w.client.PutObject(
		w.ctx,
		w.bucket,
		w.name,
		w.buf,
		int64(w.buf.Len()),
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		},
	)
	return err
}

var _ fs.StatFS = (*FS)(nil)

func (f *FS) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := f.client.StatObject(
		ctx, f.bucket, name, minio.StatObjectOptions{},
	)
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			// Check if this is a virtual directory by looking for objects
			// with this prefix
			prefix := name
			if prefix == "." {
				prefix = ""
			} else if !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}

			// List one object with this prefix to see if dir exists
			for obj := range f.client.ListObjects(
				ctx, f.bucket, minio.ListObjectsOptions{
					Prefix:    prefix,
					Recursive: false,
					MaxKeys:   1,
				},
			) {
				if obj.Err != nil {
					return nil, &fs.PathError{
						Op:   "stat",
						Path: name,
						Err:  obj.Err,
					}
				}
				// Found an object with this prefix - it's a directory
				return &s3FileInfo{
					name: path.Base(name),
					size: 0,
					mode: 0755 | fs.ModeDir,
					time: obj.LastModified,
				}, nil
			}

			// Not a file and not a directory
			return nil, &fs.PathError{
				Op:   "stat",
				Path: name,
				Err:  fs.ErrNotExist,
			}
		}
		return nil, &fs.PathError{
			Op:   "stat",
			Path: name,
			Err:  err,
		}
	}

	return &s3FileInfo{
		name: path.Base(name),
		size: info.Size,
		mode: 0644,
		time: info.LastModified,
	}, nil
}

var _ fs.ReadDirFS = (*FS)(nil)

func (f *FS) ReadDir(
	ctx context.Context, name string,
) iter.Seq2[fs.DirEntry, error] {
	return func(yield func(fs.DirEntry, error) bool) {
		prefix := name
		if prefix == "." {
			prefix = ""
		} else if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		for obj := range f.client.ListObjects(
			ctx, f.bucket, minio.ListObjectsOptions{
				Prefix:    prefix,
				Recursive: false,
			},
		) {
			if obj.Err != nil {
				yield(nil, &fs.PathError{
					Op:   "readdir",
					Path: name,
					Err:  obj.Err,
				})
				return
			}

			// Skip the directory itself
			if obj.Key == prefix {
				continue
			}

			// Extract just the name after the prefix
			relName := strings.TrimPrefix(obj.Key, prefix)
			if relName == "" {
				continue
			}

			if !yield(&s3DirEntry{
				name:  strings.TrimSuffix(relName, "/"),
				isDir: strings.HasSuffix(obj.Key, "/"),
				size:  obj.Size,
				time:  obj.LastModified,
			}, nil) {
				return
			}
		}
	}
}

var _ fs.RemoveFS = (*FS)(nil)

func (f *FS) Remove(ctx context.Context, name string) error {
	// Check if this is a virtual directory with children
	info, statErr := f.Stat(ctx, name)
	if statErr == nil && info.IsDir() {
		// Check if directory has children
		hasEntries := false
		for _, err := range f.ReadDir(ctx, name) {
			if err != nil {
				// Ignore errors - treat as empty
				break
			}
			hasEntries = true
			break
		}
		if hasEntries {
			return &fs.PathError{
				Op:   "remove",
				Path: name,
				Err:  fmt.Errorf("directory not empty"),
			}
		}
		// Empty virtual directory - nothing to remove
		return nil
	}

	// Remove the file
	err := f.client.RemoveObject(
		ctx, f.bucket, name, minio.RemoveObjectOptions{},
	)
	if err != nil {
		return &fs.PathError{
			Op:   "remove",
			Path: name,
			Err:  err,
		}
	}
	return nil
}

var _ fs.LocalizeFS = (*FS)(nil)

func (f *FS) Localize(ctx context.Context, name string) (string, error) {
	// MinIO doesn't accept "./" prefix in paths
	return strings.TrimPrefix(name, "./"), nil
}

var _ fs.AbsFS = (*FS)(nil)

func (f *FS) Abs(ctx context.Context, name string) (string, error) {
	// If already an s3:// URL, return as-is
	if strings.HasPrefix(name, "s3://") {
		return name, nil
	}

	// Resolve with WorkDir if present
	fullPath := name
	if workDir := fs.WorkDir(ctx); workDir != "" {
		fullPath = path.Join(workDir, name)
	}

	// Clean the path
	cleanPath := path.Clean(fullPath)

	// Convert to s3:// format
	if path.IsAbs(cleanPath) {
		return fmt.Sprintf("s3://%s%s", f.bucket, cleanPath), nil
	}

	// Relative path - prepend /
	return fmt.Sprintf("s3://%s/%s", f.bucket, cleanPath), nil
}
