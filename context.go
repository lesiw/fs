package fs

import "context"

type contextKey int

const (
	dirModeKey contextKey = iota
	fileModeKey
	workDirKey
)

// WithDirMode returns a context that carries a directory mode for automatic
// directory creation. When Create or WriteFile operations fail because a
// parent directory doesn't exist, and the filesystem supports MkdirFS,
// the parent directories will be created with this mode.
//
// If no directory mode is set in the context, the default mode 0755 is used.
func WithDirMode(ctx context.Context, mode Mode) context.Context {
	return context.WithValue(ctx, dirModeKey, mode)
}

// WithFileMode returns a context that carries a file mode for file creation.
// When Create or WriteFile operations create files, they use this mode.
//
// If no file mode is set in the context, the default mode 0644 is used.
func WithFileMode(ctx context.Context, mode Mode) context.Context {
	return context.WithValue(ctx, fileModeKey, mode)
}

// DirMode retrieves the directory mode from context.
// Returns 0755 if no mode is set.
func DirMode(ctx context.Context) Mode {
	if mode, ok := ctx.Value(dirModeKey).(Mode); ok {
		return mode
	}
	return 0755
}

// FileMode retrieves the file mode from context.
// Returns 0644 if no mode is set.
func FileMode(ctx context.Context) Mode {
	if mode, ok := ctx.Value(fileModeKey).(Mode); ok {
		return mode
	}
	return 0644
}

// WithWorkDir returns a context that carries a working directory for
// relative path resolution. Filesystem implementations should resolve
// relative paths relative to this directory.
//
// If no working directory is set, implementations should use their default
// working directory (typically the current working directory).
func WithWorkDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, workDirKey, dir)
}

// WorkDir retrieves the working directory from context.
// Returns an empty string if no working directory is set.
func WorkDir(ctx context.Context) string {
	if dir, ok := ctx.Value(workDirKey).(string); ok {
		return dir
	}
	return ""
}
