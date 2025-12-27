# lesiw.io/fs

[![Go Reference](https://pkg.go.dev/badge/lesiw.io/fs.svg)](https://pkg.go.dev/lesiw.io/fs)

A filesystem abstraction for Go that extends io/fs with write operations and context support.

## Features

* **Context support:** Cancellation, timeouts, and deadlines for remote filesystems.
* **Full read/write capabilities** via optional interfaces beyond io/fs's read-only model.
* **Standard io interfaces:** Returns `io.ReadCloser` and `io.WriteCloser`, not custom File types.
* **Bulk operations** with tar streams for efficient directory transfers over high-latency connections.
* **Virtual directories** to simplify writing nested paths across different storage backends.
* **Fallback implementations** provide compatibility when native operations aren't available.
* **Range-over-func iterators** for natural error handling and early termination in directory traversals.

### Feature Matrix

| Capability | io/fs | os | lesiw.io/fs |
|------------|:-----:|:--:|:-----------:|
| **Read files** | ✅ | ✅ | ✅ |
| **Write files** | ❌ | ✅ | ✅ |
| **Create/remove directories** | ❌ | ✅ | ✅ |
| **Metadata (stat, chmod)** | ✅ | ✅ | ✅ |
| **Symbolic links** | ✅ | ✅ | ✅ |
| **Standard io primitives** | ✅ | ❌ | ✅ |
| **Fallback implementations** | ✅ | ❌ | ✅ |
| **Context support** | ❌ | ❌ | ✅ |
| **Bulk operations (tar)** | ❌ | ❌ | ✅ |
| **Virtual directories** | ❌ | ❌ | ✅ |
| **Range-over-func iterators** | ❌ | ❌ | ✅ |

## Installation

```bash
go get lesiw.io/fs
```

## Quick Start

[▶️ Run this example on the Go Playground](https://go.dev/play/p/c2sE72n-j-z)

Write and read files with context support for cancellation and timeouts:

```go
package main

import (
    "context"
    "log"
    "math/rand/v2"

    "lesiw.io/fs"
    "lesiw.io/fs/memfs"
    "lesiw.io/fs/osfs"
)

var (
    ctx    = context.Background()
    fsyses = []struct {
        name string
        fn   func() fs.FS
    }{
        {"os", osfs.NewTemp},
        {"mem", memfs.New},
    }
    pick = fsyses[rand.IntN(len(fsyses))]
)

func main() {
    println("picked", pick.name)
    fsys := pick.fn()
    defer fs.Close(fsys)

    // Write a file.
    data := []byte("Hello, world!")
    if err := fs.WriteFile(ctx, fsys, "hello.txt", data); err != nil {
        log.Fatal(err)
    }

    // Read it back.
    content, err := fs.ReadFile(ctx, fsys, "hello.txt")
    if err != nil {
        log.Fatal(err)
    }
    println(string(content))

    // Output: Hello, world!
}
```

## Capabilities Are Interfaces

This package follows io/fs's philosophy: minimal core interface with optional capabilities discovered through type assertions.

The core `FS` interface requires only one method:

```go
type FS interface {
    Open(ctx context.Context, name string) (io.ReadCloser, error)
}
```

All other capabilities are optional interfaces:

```go
// Create opens a file for writing, truncating if it exists
type CreateFS interface {
    FS
    Create(ctx context.Context, name string) (io.WriteCloser, error)
}

// Mkdir creates a new directory
type MkdirFS interface {
    FS
    Mkdir(ctx context.Context, name string) error
}

// Stat returns file metadata
type StatFS interface {
    FS
    Stat(ctx context.Context, name string) (FileInfo, error)
}
```

### Building on Capabilities

Implementations can support as few or as many capabilities as make sense. Helper functions automatically check capabilities and return `ErrUnsupported` when unavailable:

```go
w, err := fs.Create(ctx, fsys, "file.txt")
if errors.Is(err, fs.ErrUnsupported) {
    // Filesystem is read-only
}
```

### Fallback Implementations

When native support is unavailable, operations may provide fallback implementations:

* `Append` falls back to reading the existing file and rewriting it with the new content appended when unsupported.
* `Rename` falls back to copying and deleting when unsupported.
* `Truncate` falls back to creating an empty file (size 0) or reading, removing, and recreating the file with adjusted size (non-zero) when unsupported.
* `ReadDir` calls `Walk` with depth 1 when unsupported.
* `Walk` recursively calls `ReadDir` when unsupported.
* `Temp` creates temporary directories with random names when `TempDirFS` is unsupported.
* Directory operations (trailing slash) use `archive/tar` when native tar commands aren't available.

These fallbacks maintain code portability across implementations while allowing native optimizations.

## Virtual Directories

Write files to nested paths without manually creating parent directories:

```go
ctx = fs.WithFileMode(ctx, 0600)
ctx = fs.WithDirMode(ctx, 0700)

// Automatically creates "logs/2025/" with mode 0700 if needed
fs.WriteFile(ctx, fsys, "logs/2025/app.log", data)
```

**Why?** Object stores like S3 use virtual directories (treating paths as object keys), while traditional filesystems require explicit directory creation. Virtual directories work seamlessly across both—traditional filesystems create parent directories with the specified mode before writing files, while object stores ignore directory creation and file modes since they're not supported.

Context carries file permissions through multiple API calls within a single operation chain—similar to request-scoped credentials or deadlines—without expanding function signatures.

## Directory Traversal

Use range-over-func iterators for natural error handling and early termination:

```go
// Walk directory tree with depth limit
for entry, err := range fs.Walk(ctx, fsys, "project", 3) {
    if err != nil {
        return err
    }
    fmt.Println(entry.Name())

    // Stop early if needed
    if entry.Name() == "stop.txt" {
        break
    }
}
```

The iterator yields entries one at a time, enabling early termination without reading entire directories and providing natural error handling within the loop.

## Bulk Operations

Directory operations use tar streams and match file operation semantics:

```go
// Read directory as tar (like reading a file)
r, err := fs.Open(ctx, fsys, "project/")
if err != nil {
    log.Fatal(err)
}
defer r.Close()
io.Copy(archiveWriter, r)

// Add files to directory (like appending to a file)
w, err := fs.Append(ctx, fsys, "project/")
if err != nil {
    log.Fatal(err)
}
defer w.Close()
io.Copy(w, newFilesArchive)

// Empty directory (like truncating a file to zero)
err = fs.Truncate(ctx, fsys, "project/", 0)

// Replace directory contents (like creating/truncating a file)
w, err = fs.Create(ctx, fsys, "restore/")
if err != nil {
    log.Fatal(err)
}
defer w.Close()
io.Copy(w, archiveReader)
```

**Why?** Remote filesystems benefit from bulk operations—transferring many small files individually is slow. The trailing slash convention clearly indicates directory operations while matching the semantics users already understand from file operations.

Optional interfaces enable native implementations:
- `DirFS` - Read directories as tar (useful for read-only filesystems)
- `AppendDirFS` - Write tar streams to directories
- `TruncateDirFS` - Efficiently empty directories

When not implemented, operations automatically fall back to walking the filesystem and using `archive/tar`.

## Example Implementations

Reference implementations demonstrating the abstraction across diverse backends:

* [HTTP](../internal/example/http) - Read-only HTTP filesystem
* [S3](../internal/example/s3) - Amazon S3 via MinIO SDK
* [SFTP](../internal/example/sftp) - SSH File Transfer Protocol
* [SMB](../internal/example/smb) - SMB/CIFS network shares
* [SSH](../internal/example/ssh) - SSH with tar for bulk operations
* [WebDAV](../internal/example/webdav) - WebDAV protocol

These implementations require Docker to run tests.

## Testing

The `lesiw.io/fs/fstest` package provides a test suite for filesystem implementations:

```go
func TestMyFS(t *testing.T) {
    fsys := myfs.New(...)
    t.Cleanup(func() { fsys.Close() })
    fstest.TestFS(t.Context(), t, fsys)
}
```

The test suite automatically detects capabilities through type assertions and validates all supported operations.

## Documentation

Full documentation available at [pkg.go.dev/lesiw.io/fs](https://pkg.go.dev/lesiw.io/fs).

## License

See LICENSE file in the repository root.
