package fs

import (
	"context"
	"errors"
	"path"
	"strings"
)

// A RelFS is a file system with the Rel method.
type RelFS interface {
	FS

	// Rel returns a relative path that is lexically equivalent to targpath
	// when joined to basepath with an intervening separator.
	//
	// This is a pure lexical operation. An error is returned if targpath
	// can't be made relative to basepath.
	Rel(ctx context.Context, basepath, targpath string) (string, error)
}

// Rel returns a relative path that is lexically equivalent to targpath when
// joined to basepath with an intervening separator. That is,
// [path.Join](basepath, Rel(basepath, targpath)) is equivalent to targpath
// itself. On success, the returned path will always be relative to basepath,
// even if basepath and targpath share no elements.
//
// Analogous to: [filepath.Rel], realpath --relative-to.
//
// An error is returned if targpath can't be made relative to basepath.
//
// Rel is a pure lexical operation using the path package (forward slashes).
// It calls [path.Clean] on the result.
func Rel(
	ctx context.Context, fsys FS, basepath, targpath string,
) (string, error) {
	// Try native capability first
	if rfs, ok := fsys.(RelFS); ok {
		rel, err := rfs.Rel(ctx, basepath, targpath)
		if !errors.Is(err, ErrUnsupported) {
			return rel, err
		}
	}

	// Fallback: pure lexical operation using path package
	return rel(basepath, targpath)
}

// rel is a pure lexical implementation of Rel using path package conventions.
// This is adapted from filepath.Rel but uses forward slashes consistently.
func rel(basepath, targpath string) (string, error) {
	base := path.Clean(basepath)
	targ := path.Clean(targpath)

	if targ == base {
		return ".", nil
	}

	if base == "." {
		base = ""
	}

	// Can't make relative if one is absolute and the other isn't
	baseSlashed := strings.HasPrefix(base, "/")
	targSlashed := strings.HasPrefix(targ, "/")
	if baseSlashed != targSlashed {
		return "", errors.New(
			"Rel: can't make " + targpath + " relative to " + basepath,
		)
	}

	// Position base[b0:bi] and targ[t0:ti] at the first differing elements
	bl := len(base)
	tl := len(targ)
	var b0, bi, t0, ti int
	for {
		for bi < bl && base[bi] != '/' {
			bi++
		}
		for ti < tl && targ[ti] != '/' {
			ti++
		}
		if targ[t0:ti] != base[b0:bi] {
			break
		}
		if bi < bl {
			bi++
		}
		if ti < tl {
			ti++
		}
		b0 = bi
		t0 = ti
	}

	if base[b0:bi] == ".." {
		return "", errors.New(
			"Rel: can't make " + targpath + " relative to " + basepath,
		)
	}

	if b0 != bl {
		// Base elements left. Must go up before going down.
		seps := strings.Count(base[b0:bl], "/")
		size := 2 + seps*3
		if tl != t0 {
			size += 1 + tl - t0
		}
		buf := make([]byte, size)
		n := copy(buf, "..")
		for range seps {
			buf[n] = '/'
			copy(buf[n+1:], "..")
			n += 3
		}
		if t0 != tl {
			buf[n] = '/'
			copy(buf[n+1:], targ[t0:])
		}
		return string(buf), nil
	}

	return targ[t0:], nil
}
