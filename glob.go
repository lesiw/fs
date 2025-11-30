package fs

import (
	"context"
	"errors"

	"lesiw.io/fs/path"
)

// A GlobFS is a file system with the Glob method.
//
// If not implemented, Glob falls back to pattern matching using
// StatFS and ReadDirFS.
type GlobFS interface {
	FS

	// Glob returns the names of all files matching pattern.
	// The pattern syntax is the same as in [path.Match].
	Glob(ctx context.Context, pattern string) ([]string, error)
}

// Glob returns the names of all files matching pattern.
// Analogous to: [io/fs.Glob], [path.Match], glob, find, 9P walk.
//
// The pattern syntax is the same as in [path.Match]. The pattern may
// describe hierarchical names such as usr/*/bin/ed.
//
// Glob ignores file system errors such as I/O errors reading directories.
// The only possible returned error is [path.ErrBadPattern], reporting that
// the pattern is malformed.
//
// Requires: [GlobFS] ||
// ([StatFS] && ([ReadDirFS] || [WalkFS]))
func Glob(ctx context.Context, fsys FS, pattern string) ([]string, error) {
	if gfs, ok := fsys.(GlobFS); ok {
		matches, err := gfs.Glob(ctx, pattern)
		if err != nil && !errors.Is(err, ErrUnsupported) {
			return matches, err
		}
		if err == nil {
			return matches, nil
		}
		// Fall through to fallback if ErrUnsupported
	}

	// Check if fallback is possible - requires StatFS and ReadDirFS
	_, hasStat := fsys.(StatFS)
	_, hasReadDir := fsys.(ReadDirFS)

	if !hasStat || !hasReadDir {
		return nil, &PathError{
			Op:   "glob",
			Path: pattern,
			Err:  ErrUnsupported,
		}
	}

	return globWithLimit(ctx, fsys, pattern, 0)
}

func globWithLimit(
	ctx context.Context, fsys FS, pattern string, depth int,
) (matches []string, err error) {
	// This limit is added to prevent stack exhaustion issues.
	// See CVE-2022-30630.
	const pathSeparatorsLimit = 10000
	if depth > pathSeparatorsLimit {
		return nil, path.ErrBadPattern
	}

	// Check pattern is well-formed.
	if _, err := path.Match(pattern, ""); err != nil {
		return nil, err
	}
	if !hasMeta(pattern) {
		if _, err = Stat(ctx, fsys, pattern); err != nil {
			return nil, nil
		}
		return []string{pattern}, nil
	}

	dir, file := path.Split(pattern)
	// Our Split already returns clean dir without trailing separator
	if dir == "" {
		dir = "."
	}

	if !hasMeta(dir) {
		return glob(ctx, fsys, dir, file, nil)
	}

	// Prevent infinite recursion. See issue 15879.
	if dir == pattern {
		return nil, path.ErrBadPattern
	}

	var m []string
	m, err = globWithLimit(ctx, fsys, dir, depth+1)
	if err != nil {
		return nil, err
	}
	for _, d := range m {
		matches, err = glob(ctx, fsys, d, file, matches)
		if err != nil {
			return
		}
	}
	return
}

// glob searches for files matching pattern in the directory dir
// and appends them to matches, returning the updated slice.
// If the directory cannot be opened, glob returns the existing matches.
// New matches are added in lexicographical order.
func glob(
	ctx context.Context, fsys FS, dir, pattern string, matches []string,
) (m []string, e error) {
	m = matches

	// Read directory using ReadDir
	for info, err := range ReadDir(ctx, fsys, dir) {
		if err != nil {
			return m, nil // ignore I/O error
		}
		n := info.Name()
		matched, matchErr := path.Match(pattern, n)
		if matchErr != nil {
			return m, matchErr
		}
		if matched {
			m = append(m, path.Join(dir, n))
		}
	}
	return
}

// hasMeta reports whether path contains any of the magic characters
// recognized by path.Match.
func hasMeta(p string) bool {
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '*', '?', '[', '\\':
			return true
		}
	}
	return false
}
