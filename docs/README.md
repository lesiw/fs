# lesiw.io/fs

[![Go Reference](https://pkg.go.dev/badge/lesiw.io/fs.svg)](https://pkg.go.dev/lesiw.io/fs)
[![CI](https://github.com/lesiw/fs/actions/workflows/main.yml/badge.svg?branch=main)](https://github.com/lesiw/fs/actions/workflows/main.yml)
[![License](https://img.shields.io/github/license/lesiw/fs)](../LICENSE)

A filesystem abstraction for Go that extends `io/fs` with write
operations and context support. The same code reads and writes local
disks, in-memory filesystems, and remote systems reached over SSH,
S3, SMB, or WebDAV.

```go
// Copying between filesystems is io.Copy of standard interfaces.
w, err := fs.Create(ctx, remote, "backup.tar.gz")
if err != nil {
    return err
}
defer w.Close()
r, err := fs.Open(ctx, local, "backup.tar.gz")
if err != nil {
    return err
}
defer r.Close()
_, err = io.Copy(w, r)
```

[API reference](https://pkg.go.dev/lesiw.io/fs) ·
[Source](https://github.com/lesiw/fs)

## The model

1. **`FS` is one method.** `Open(ctx, name) (io.ReadCloser, error)`.
   Files are standard `io` values, never a custom `File` type.
2. **Every other capability is an optional interface** discovered
   by type assertion, in the manner of `io/fs`.
3. **Helper functions check capabilities and fall back**, so portable
   code calls one function either way.
4. **The context carries the operation's ambience** — cancellation
   and timeouts, file and directory modes, the working directory.
5. **A trailing slash means a directory, and directories are tar
   streams.**

## Feature matrix

| Capability | io/fs | os | lesiw.io/fs |
|------------|:-----:|:--:|:-----------:|
| **Read files** | ✅ | ✅ | ✅ |
| **Write files** | ❌ | ✅ | ✅ |
| **Create/remove directories** | ❌ | ✅ | ✅ |
| **Read metadata (stat)** | ✅ | ✅ | ✅ |
| **Write metadata (chmod, chtimes)** | ❌ | ✅ | ✅ |
| **Create symbolic links** | ❌ | ✅ | ✅ |
| **io interfaces, not File types** | ❌ | ❌ | ✅ |
| **Fallback implementations** | ✅ | ❌ | ✅ |
| **Context support** | ❌ | ❌ | ✅ |
| **Bulk operations (tar)** | ❌ | ❌ | ✅ |
| **Virtual directories** | ❌ | ❌ | ✅ |
| **Range-over-func iterators** | ❌ | ❌ | ✅ |

## Install

```sh
go get lesiw.io/fs
```

Requires Go 1.24.2 or later.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "lesiw.io/fs"
    "lesiw.io/fs/memfs"
)

func main() {
    ctx, fsys := context.Background(), memfs.New()
    defer fs.Close(fsys)

    err := fs.WriteFile(ctx, fsys, "hello.txt", []byte("Hello, world!"))
    if err != nil {
        log.Fatal(err)
    }

    content, err := fs.ReadFile(ctx, fsys, "hello.txt")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(content))
}
```

> [!TIP]
> The in-memory filesystem runs in the Go Playground, so the package
> can be tried without installing anything:
> [run an example](https://go.dev/play/p/c2sE72n-j-z).

## Capabilities are interfaces

The core interface requires one method.

```go
type FS interface {
    Open(ctx context.Context, name string) (io.ReadCloser, error)
}
```

Everything else is an optional interface an implementation may add —
`CreateFS` for writing, `MkdirFS` for directories, `StatFS` for
metadata, two dozen in all. Helper functions probe for the
capability and report `fs.ErrUnsupported` when an operation has no
path forward.

```go
w, err := fs.Create(ctx, fsys, "file.txt")
if errors.Is(err, fs.ErrUnsupported) {
    // Filesystem is read-only.
}
```

Many helpers carry fallbacks, so an implementation supplies the
operations it can do natively and inherits the rest. `Append` falls
back to read-and-rewrite, `Rename` to copy-and-delete, `Walk` and
`ReadDir` to each other, and directory-as-tar operations to
`archive/tar` over a filesystem walk.

## Context values

Cancellation and deadlines work the way they do everywhere else in
Go, which matters most when the filesystem is on the far side of a
network.

```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
data, err := fs.ReadFile(ctx, remote, "large-file.dat")
```

File and directory modes travel on the context too, where they apply
to every operation in a chain — including parent directories created
implicitly along the way.

```go
ctx = fs.WithFileMode(ctx, 0600)
ctx = fs.WithDirMode(ctx, 0700)
err := fs.WriteFile(ctx, fsys, "logs/2026/app.log", data)
```

That call creates `logs/2026/` with mode 0700 when it doesn't exist,
then writes the file with mode 0600. Object stores treat the nested
path as a key and skip directory creation; traditional filesystems
create the parents. The calling code is the same for both.

## Directories are tar streams

A trailing slash names a directory, and directories read and write
as tar archives with the same verbs as files.

```go
// Read a directory as an archive, like reading a file.
r, err := fs.Open(ctx, fsys, "project/")

// Add files to a directory, like appending to a file.
w, err := fs.Append(ctx, fsys, "project/")

// Empty a directory, like truncating a file.
err = fs.Truncate(ctx, fsys, "project/", 0)

// Replace a directory's contents, like creating a file.
w, err = fs.Create(ctx, fsys, "restore/")
```

Implementations with native tar support (`DirFS`, `AppendDirFS`,
`TruncateDirFS`) move one stream instead of one request per file;
without them, the helpers fall back to a filesystem walk with
`archive/tar`.

## Directory traversal

`Walk` and `ReadDir` are range-over-func iterators, so errors are
handled in the loop and early termination costs nothing.

```go
for entry, err := range fs.Walk(ctx, fsys, "project", 3) {
    if err != nil {
        return err
    }
    if entry.Name() == "target.txt" {
        return process(entry)
    }
}
```

## Implementations

Maintained in this module:

- [osfs](https://pkg.go.dev/lesiw.io/fs/osfs) — the local filesystem,
  backed by `os`, implementing most optional interfaces natively
- [memfs](https://pkg.go.dev/lesiw.io/fs/memfs) — in-memory,
  Playground-safe, useful in tests and examples

Reference implementations demonstrating the abstraction across
diverse backends, with Docker-based tests:

- [HTTP](../internal/example/http) — read-only HTTP filesystem
- [S3](../internal/example/s3) — Amazon S3 via the MinIO SDK
- [SFTP](../internal/example/sftp) — SSH File Transfer Protocol
- [SMB](../internal/example/smb) — SMB/CIFS network shares
- [SSH](../internal/example/ssh) — remote filesystem over SSH
- [WebDAV](../internal/example/webdav) — WebDAV protocol

[lesiw.io/command](https://github.com/lesiw/command) builds on this
package: its `command.FS` derives a filesystem for any machine it
can run commands on, ssh hosts and containers included.

## Testing

The [fstest](https://pkg.go.dev/lesiw.io/fs/fstest) package validates
filesystem implementations.

```go
func TestMyFS(t *testing.T) {
    fsys := myfs.New()
    t.Cleanup(func() { _ = fs.Close(fsys) })
    fstest.TestFS(t.Context(), t, fsys)
}
```

The suite detects capabilities through type assertions and exercises
every operation the implementation supports, fallbacks included.

## Documentation

- [pkg.go.dev/lesiw.io/fs](https://pkg.go.dev/lesiw.io/fs) — full API
  documentation
- [cmdbuf.io](https://cmdbuf.github.io) — automation built on this
  package and lesiw.io/command, with worked examples

## License

[BSD 3-Clause](../LICENSE)
