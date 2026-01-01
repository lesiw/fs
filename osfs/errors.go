package osfs

import "syscall"

// errNotDir is the underlying syscall error for "not a directory".
// This is used to translate OS-specific errors to fs.ErrNotDir.
var errNotDir error = syscall.ENOTDIR
