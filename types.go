// Package fs provides a filesystem abstraction for local and remote
// filesystems.
//
// Package fs is inspired by both [io/fs] and [os]. It follows io/fs's
// philosophy of minimal core interfaces with optional capabilities, but
// extends it with
// write operations and requires [context.Context] for all operations. The core
// [FS] interface requires only a single method, Open, which returns a standard
// [io.ReadCloser]. All other capabilities are optional and discovered through
// type assertions.
//
// Every operation accepts a context.Context as the first parameter, enabling
// cancellation of long-running transfers, timeouts for remote operations, and
// request-scoped values like file modes. This is critical for network
// filesystems (SSH, S3, WebDAV) but local filesystem implementations may
// ignore context while still accepting it for interface compatibility.
//
// Example with timeout:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	data, err := fs.ReadFile(ctx, remoteFS, "large-file.dat")
//
// # Path Handling
//
// Path operations are primarily lexical, handled by the [lesiw.io/fs/path]
// subpackage. This package auto-detects path styles (Unix, Windows, URLs) and
// applies appropriate rules without accessing the filesystem.
//
//	import "lesiw.io/fs/path"
//
//	path.Join("foo", "bar")              // "foo/bar" (Unix-style)
//	path.Join("C:\\", "Users", "foo")    // "C:\Users\foo" (Windows-style)
//	path.Join("https://example.com", "api") // "https://example.com/api"
//
// For the rare cases requiring OS-specific path resolution, use [Localize]:
//
//	// Converts "dir/file.txt" to "dir\file.txt" on Windows
//	resolved, err := fs.Localize(ctx, fsys, "dir/file.txt")
//
// Operations like [Open] and [Create] automatically call [Localize] when the
// filesystem implements [LocalizeFS], so explicit calls are rarely needed.
//
// A trailing slash indicates a directory path. [Create]("foo") creates a file
// named "foo" and opens it for writing, while [Create]("foo/") creates a
// directory named "foo" and opens it for writing as a tar stream. This
// convention applies to [Open], [Create], [Append], and [Truncate].
//
// # Virtual Directories
//
// Some filesystems use virtual directories that don't physically exist. Object
// stores like S3 commonly use this pattern, treating paths as object keys
// rather than traditional directories.
//
// For these filesystems, implementations can automatically create parent
// directories when writing files. This enables writing to nested paths without
// explicit directory creation:
//
//	// Automatically creates "logs/2025/" if needed
//	ctx = fs.WithDirMode(ctx, 0755)
//	fs.WriteFile(ctx, fsys, "logs/2025/app.log", data)
//
// Object stores typically don't need this feature because they can create
// files at arbitrarily nested paths without directory creation. However,
// the virtual directory pattern works across both traditional filesystems and
// object stores,
// simplifying code portability.
//
// # Optional Interfaces
//
// The core FS interface is read-only, requiring only Open which returns
// [io.ReadCloser]. Write operations and other capabilities are provided
// through optional interfaces. Unlike [io/fs] which defines a custom File
// type, this
// package uses standard Go interfaces like [io.ReadCloser] and
// [io.WriteCloser],
// maximizing composability with existing code. Files may also implement
// [io.Seeker], [io.ReaderAt], [io.WriterAt], or other standard interfaces
// depending on filesystem capabilities.
//
// Optional write and metadata interfaces:
//
//   - [AbsFS] - Absolute path resolution
//   - [AppendDirFS] - Write tar streams to directories
//   - [AppendFS] - Append to existing files
//   - [ChmodFS] - Change file permissions
//   - [ChownFS] - Change file ownership
//   - [ChtimesFS] - Change file timestamps
//   - [CreateFS] - Create or truncate files for writing
//   - [DirFS] - Read directories as tar streams
//   - [GlobFS] - Pattern-based file matching
//   - [LocalizeFS] - OS-specific path formatting
//   - [MkdirFS] - Create directories
//   - [ReadDirFS] - List directory contents
//   - [ReadLinkFS] - Read symlink targets and stat without following
//   - [RelFS] - Relative path computation
//   - [RemoveAllFS] - Recursively delete directories
//   - [RemoveFS] - Delete files and empty directories
//   - [RenameFS] - Move or rename files
//   - [StatFS] - Query file metadata
//   - [SymlinkFS] - Create symbolic links
//   - [TempDirFS] - Native temporary directory support
//   - [TempFS] - Native temporary file support
//   - [TruncateDirFS] - Efficiently empty directories
//   - [TruncateFS] - Change file size
//   - [WalkFS] - Efficient directory traversal
//
// Helper functions check capabilities automatically and return
// [ErrUnsupported]
// when an operation isn't available.
//
//	w, err := fs.Create(ctx, fsys, "file.txt")
//	if errors.Is(err, fs.ErrUnsupported) {
//	    // Filesystem doesn't support writing
//	}
//
// # File Modes via Context
//
// File permissions can be set via context and apply throughout the operation
// chain. This is particularly useful with implicit directory creation, where
// multiple operations may occur (creating parent directories, then the file).
//
//	ctx = fs.WithFileMode(ctx, 0600)  // Private files
//	ctx = fs.WithDirMode(ctx, 0700)   // Private directories
//
//	// All operations use these modes
//	fs.Create(ctx, fsys, "secret.key")        // Creates file with mode 0600
//	fs.MkdirAll(ctx, fsys, "private/data")    // Creates dirs with mode 0700
//
// Default modes are 0644 for files and 0755 for directories. Different call
// chains can use different modes simultaneously by deriving separate contexts.
//
// File modes are request-scoped values that pass through multiple API calls
// (Create, Mkdir, MkdirAll) within a single operation chain. This makes
// context the natural place to store them, similar to how request-scoped
// credentials or deadlines flow through API boundaries.
//
// For filesystems that don't support file permissions (like many object
// stores),
// these values are ignored.
//
// # Bulk Operations
//
// Directory operations use tar streams and match file operation semantics. A
// trailing slash indicates a directory path. This is valuable for remote
// filesystems where transferring many small files individually would be slow.
//
//	// Read directory as tar (like reading a file)
//	r, err := fs.Open(ctx, fsys, "project/")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	io.Copy(archiveWriter, r)
//	r.Close()
//
//	// Add files to directory (like appending to a file)
//	w, err := fs.Append(ctx, fsys, "project/")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	io.Copy(w, newFilesArchive)
//	w.Close()
//
//	// Empty directory (like truncating a file to zero)
//	err = fs.Truncate(ctx, fsys, "project/", 0)
//
//	// Replace directory contents (truncate + append)
//	w, err = fs.Create(ctx, fsys, "restore/")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	io.Copy(w, archiveReader)
//	w.Close()
//
// Optional interfaces enable native tar implementations: [DirFS] for reading,
// [AppendDirFS] for writing, and [TruncateDirFS] for emptying. When not
// implemented, operations fall back to walking the filesystem and using
// [archive/tar]. Implementations should use native tar commands when
// available.
// For implementations that natively support other archive formats (like zip),
// consider writing stream converters to and from tar format.
//
// # Fallback Implementations
//
// Many operations provide fallback implementations when native support is
// unavailable.
//
//   - [Append] falls back to reading the existing file and rewriting it with
//     new content appended when [AppendFS] is not implemented.
//   - [Rename] falls back to copying and deleting when [RenameFS] is not
//     implemented.
//   - [Truncate] falls back to creating an empty file when size is 0, or
//     reading, removing, and recreating the file with adjusted size for
//     non-zero sizes, when [TruncateFS] is not implemented.
//   - [ReadDir] calls [Walk] with depth 1 when [ReadDirFS] is not implemented.
//   - [Walk] recursively calls [ReadDir] when [WalkFS] is not implemented.
//   - [Temp] creates temporary files or directories with random names when
//     [TempFS] or [TempDirFS] are not implemented.
//
// These operations succeed regardless of native support. Native
// implementations
// should be added when more efficient approaches are available for specific
// operations.
//
// # Range-Over-Func Iterators
//
// ReadDir and Walk use Go 1.23+ range-over-func iterators for directory
// traversal.
//
//	for entry, err := range fs.ReadDir(ctx, fsys, "dir") {
//	    if err != nil {
//	        return err
//	    }
//	    fmt.Println(entry.Name())
//	}
//
// This enables early termination without reading entire directories and
// provides
// natural error handling within the loop.
//
// # Testing
//
// The [lesiw.io/fs/fstest] package provides a test suite for filesystem
// implementations.
//
//	func TestMyFS(t *testing.T) {
//	    fsys := myfs.New(...)
//	    t.Cleanup(func() { fsys.Close() })
//	    fstest.TestFS(t.Context(), t, fsys)
//	}
//
// The test suite automatically detects capabilities through type assertions
// and
// validates all supported operations.
//
// # Example Implementations
//
// The internal/example directory contains reference implementations
// demonstrating the abstraction across diverse backends:
//
//   - internal/example/http - Read-only HTTP filesystem
//   - internal/example/s3 - Amazon S3 via MinIO SDK
//   - internal/example/sftp - SSH File Transfer Protocol
//   - internal/example/smb - SMB/CIFS network shares
//   - internal/example/ssh - SSH with tar for bulk operations
//   - internal/example/webdav - WebDAV protocol
//
// These implementations require Docker to run tests.
//
// # Compatibility
//
// This package is not a drop-in replacement for [io/fs] or [os] because every
// operation requires a context.Context parameter. However, the design
// philosophy
// is closely aligned with io/fs: minimal core interface, optional capabilities
// through type assertions, and reliance on standard Go interfaces.
//
// The package follows a conservative design that will only be modified
// carefully
// over time. New capabilities may be added based on how broadly useful and
// applicable they are across different filesystem implementations.
//
// The [lesiw.io/fs/osfs] subpackage provides a maintained reference
// implementation
// backed by the [os] package, demonstrating all optional interfaces
package fs

import (
	"errors"
	"io/fs"
)

// DirEntry describes a directory entry.
//
// This interface extends the standard io/fs.DirEntry with path information.
// Path() returns the full path of the entry when called on entries returned
// by Walk. For entries returned by ReadDir, Path() returns an empty string
// since ReadDir only provides entries within a single directory without
// path context.
type DirEntry interface {
	// Name returns the name of the file (or subdirectory) described
	// by the entry. This name is only the final element of the path
	// (the base name), not the entire path.
	Name() string

	// IsDir reports whether the entry describes a directory.
	IsDir() bool

	// Type returns the type bits for the entry. The type bits are a
	// subset of the usual FileMode bits, those returned by the
	// FileMode.Type method.
	Type() fs.FileMode

	// Info returns the FileInfo for the file or subdirectory
	// described by the entry. The returned FileInfo may be from the
	// time of the original directory read or from the time of the
	// call to Info. If the file has been removed or renamed since
	// the directory read, Info may return an error satisfying
	// errors.Is(err, ErrNotExist). If the entry denotes a symbolic
	// link, Info reports the information about the link itself, not
	// the link's target.
	Info() (FileInfo, error)

	// Path returns the full path of this entry relative to the walk root.
	// For entries returned by ReadDir, this returns an empty string.
	// For entries returned by Walk, this returns the full path.
	Path() string
}

// A FileInfo describes a file and is returned by [Stat].
type FileInfo = fs.FileInfo

// A Mode represents a file's mode and permission bits.
// The bits have the same definition on all systems, so that
// information about files can be moved from one system
// to another portably. Not all bits apply to all systems.
// The only required bit is [ModeDir] for directories.
type Mode = fs.FileMode

// PathError records an error and the operation and file path that caused it.
type PathError = fs.PathError

// newPathError creates a PathError if err is not nil, otherwise returns nil.
// This is useful for wrapping errors while preserving nil returns.
func newPathError(op, path string, err error) error {
	if err == nil {
		return nil
	}
	return &PathError{Op: op, Path: path, Err: err}
}

// Generic file system errors.
var (
	ErrInvalid     = fs.ErrInvalid
	ErrPermission  = fs.ErrPermission
	ErrExist       = fs.ErrExist
	ErrNotExist    = fs.ErrNotExist
	ErrClosed      = fs.ErrClosed
	ErrUnsupported = errors.ErrUnsupported
)

// Valid values for [Mode].
//
//ignore:linelen
const (
	// The single letters are the abbreviations
	// used by the String method's formatting.
	ModeDir        = fs.ModeDir        // d: is a directory
	ModeAppend     = fs.ModeAppend     // a: append-only
	ModeExclusive  = fs.ModeExclusive  // l: exclusive use
	ModeTemporary  = fs.ModeTemporary  // T: temporary file; Plan 9 only
	ModeSymlink    = fs.ModeSymlink    // L: symbolic link
	ModeDevice     = fs.ModeDevice     // D: device file
	ModeNamedPipe  = fs.ModeNamedPipe  // p: named pipe (FIFO)
	ModeSocket     = fs.ModeSocket     // S: Unix domain socket
	ModeSetuid     = fs.ModeSetuid     // u: setuid
	ModeSetgid     = fs.ModeSetgid     // g: setgid
	ModeCharDevice = fs.ModeCharDevice // c: Unix character device, when ModeDevice is set
	ModeSticky     = fs.ModeSticky     // t: sticky
	ModeIrregular  = fs.ModeIrregular  // ?: non-regular file; nothing else is known about this file

	// Mask for the type bits. For regular files, none will be set.
	ModeType = fs.ModeType

	ModePerm = fs.ModePerm // Unix permission bits
)
