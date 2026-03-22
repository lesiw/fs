// Package path implements utility routines for manipulating filesystem paths
// in a lexical manner, supporting Unix, Windows, and URL path styles.
//
// Unlike the standard library's path package (Unix-only) and filepath package
// (OS-specific), this package automatically detects the path style and applies
// appropriate rules:
//
//   - Unix-style: Forward slashes, single root /
//   - Windows-style: Backslashes, drive letters (C:\, D:\, etc.)
//   - URL-style: Forward slashes, protocol://host/ roots
//
// All path operations are purely lexical. In particular, they do not access
// the filesystem or account for the effect of symbolic links, mount points,
// or other filesystem-specific behavior.
//
// Trailing slashes indicate directories:
//
//	path.IsDir("foo/bar/")   // true
//	path.IsDir("foo/bar")    // false
package path

import (
	stdpath "path"
	"strings"
)

// Join joins path elements into a single path, detecting the path style
// (Unix, Windows, or URL) from the first element and applying appropriate
// rules. Empty elements are ignored, except if the last element is empty,
// which adds a trailing separator to indicate a directory.
//
// The path style is determined solely by the first element. Each
// subsequent element is independently style-detected and split by
// its own separator. Absolute elements in non-first positions are
// treated as literal segments — their root is ignored.
//
// Examples:
//
//	Join("foo", "bar")                     // "foo/bar"
//	Join("C:\\", "foo", "bar")             // "C:\foo\bar"
//	Join("https://example.com", "foo")     // "https://example.com/foo"
//	Join("foo", "bar", "")                 // "foo/bar/"
func Join(elem ...string) string {
	if len(elem) == 0 {
		return ""
	}

	style := detectStyle(elem[:1])

	// First element: split normally to extract root/prefix.
	var parts []string
	if elem[0] != "" {
		dir, file := Split(elem[0])
		if dir != "" {
			parts = append(parts, splitAll(dir)...)
		}
		// Skip empty file (from trailing separator) when more elements
		// follow. The trailing separator is meaningless when joining
		// additional segments, and the empty string would create a
		// double separator that can be misdetected as a URL scheme.
		if file != "" || len(elem) == 1 {
			parts = append(parts, file)
		}
	}

	// Subsequent elements: split using native Split/splitAll, which
	// handle style detection internally. Absolute elements are taken
	// as literal segments — their root is ignored, not merged.
	for _, e := range elem[1:] {
		if e == "" {
			parts = append(parts, "")
			continue
		}
		if IsAbs(e) {
			parts = append(parts, e)
			continue
		}
		dir, file := Split(e)
		if dir != "" {
			parts = append(parts, splitAll(dir)...)
		}
		parts = append(parts, file)
	}

	return Clean(joinParts(parts, style))
}

// Split splits path into directory and file components.
// The directory does not include a trailing separator, except for roots
// and local prefixes (./ or .\) which preserve the path style.
// Returns ("", file) if path has no directory component.
// Returns (dir, "") if path ends with a trailing separator (is a directory).
func Split(path string) (dir, file string) {
	style := detectStyle([]string{path})
	sep := string(style.sep)
	local := "." + sep

	// Handle trailing separator (directory)
	if strings.HasSuffix(path, sep) {
		if isRoot(path, style) || path == local {
			return path, ""
		}
		return strings.TrimSuffix(path, sep), ""
	}

	// For URL-style paths, skip the :// when finding the last separator
	searchStart := 0
	if style.kind == styleURL {
		if protoEnd := strings.Index(path, "://"); protoEnd >= 0 {
			searchStart = protoEnd + 3
		}
	}

	// Find last separator (after searchStart for URLs)
	i := strings.LastIndex(path[searchStart:], sep)
	if i < 0 {
		if isRoot(path, style) {
			return path, ""
		}
		return "", path
	}
	i += searchStart

	dir = path[:i+1]
	file = path[i+1:]

	// Roots and local prefixes keep their trailing separator
	if isRoot(dir, style) || dir == local {
		return dir, addDot(file, style)
	}

	return strings.TrimSuffix(dir, sep), addDot(file, style)
}

// addDot prefixes file with a local style marker (./ or .\) when
// it contains a separator from another style. Without this, Join's
// per-element style detection would misinterpret the file's style.
func addDot(file string, style pathStyle) string {
	if file == "" {
		return file
	}
	if dir, f := Split(file); dir != "" || f != file {
		return "." + string(style.sep) + file
	}
	return file
}

// Base returns the last element of path.
// Returns "" if path has a trailing separator (directory).
func Base(path string) string {
	_, file := Split(path)
	return file
}

// Dir returns the directory containing path.
// Returns "" if path has no directory component.
func Dir(path string) string {
	dir, _ := Split(path)
	return dir
}

// IsDir reports whether the path is lexically a directory.
// A path is a directory if it has a trailing separator.
func IsDir(path string) bool {
	if path == "" {
		return false
	}
	_, file := Split(path)
	return file == ""
}

// IsRoot reports whether the path is lexically a root.
// Roots include:
//   - "/" (Unix root)
//   - "C:\", "D:\" etc. (Windows drive roots)
//   - "https://example.com/", "s3://bucket/" etc. (URL roots)
func IsRoot(path string) bool {
	style := detectStyle([]string{path})
	return isRoot(path, style)
}

// IsAbs reports whether the path is lexically absolute.
// Absolute paths include:
//   - Paths starting with "/" (Unix-style)
//   - Paths starting with drive letter (C:\, D:\, etc.) (Windows-style)
//   - Paths starting with protocol:// (https://, s3://, etc.) (URL-style)
func IsAbs(path string) bool {
	if path == "" {
		return false
	}

	// Unix-style: starts with /
	if path[0] == '/' {
		return true
	}

	// Windows-style: starts with [letter]:\
	if len(path) >= 3 && isDriveLetter(path[0]) && path[1] == ':' &&
		(path[2] == '\\' || path[2] == '/') {
		return true
	}

	// URL-style: contains :// with a non-empty protocol
	if idx := strings.Index(path, "://"); idx > 0 {
		return true
	}

	return false
}

// Clean returns the canonical path name equivalent to path by purely lexical
// processing. It applies the following rules iteratively until no further
// processing can be done:
//
//  1. Replace multiple separators with a single separator
//  2. Eliminate redundant . path elements
//  3. Eliminate .. path elements and the preceding element
//  4. Preserve trailing separator (indicates directory)
//  5. Add leading ./ or .\ to relative paths (preserves detected style)
//
// If .. would escape a root, Clean stops at the root (e.g., "/.." becomes "/",
// "C:\.." becomes "C:\").
func Clean(path string) string {
	if path == "" {
		return "."
	}

	style := detectStyle([]string{path})
	sep := string(style.sep)

	// Preserve trailing separator
	hadTrailing := strings.HasSuffix(path, sep)

	// Check for and preserve leading ./ or .\
	var localPrefix string
	if strings.HasPrefix(path, "./") || strings.HasPrefix(path, ".\\") {
		localPrefix = "." + sep
		path = strings.TrimLeft(path[2:], sep)
	}

	// Split into parts, preserving special prefixes
	var prefix string
	var parts []string

	if style.kind == styleURL {
		// For URLs, extract protocol://host/ as a single "root" part
		protoEnd := strings.Index(path, "://")
		if protoEnd >= 0 {
			// Find the first / after ://
			hostStart := protoEnd + 3
			hostEnd := strings.Index(path[hostStart:], "/")
			if hostEnd < 0 {
				// No path after host — normalize to include trailing /
				prefix = path + "/"
			} else {
				prefix = path[:hostStart+hostEnd+1] // Include the /
				rest := path[hostStart+hostEnd+1:]
				if rest != "" {
					parts = strings.Split(rest, sep)
				}
			}
		} else {
			parts = strings.Split(path, sep)
		}
	} else if style.kind == styleWindows {
		// For Windows, preserve drive letter (only at start
		// of path, not after .\ prefix).
		if localPrefix == "" && len(path) >= 2 &&
			path[1] == ':' && isDriveLetter(path[0]) {
			if len(path) >= 3 && path[2:3] == sep {
				prefix = path[:3] // C:\
				rest := path[3:]
				if rest != "" {
					parts = strings.Split(rest, sep)
				}
			} else {
				prefix = path[:2] + sep // C: -> C:\
				rest := path[2:]
				if rest != "" {
					rest = strings.TrimPrefix(rest, sep)
					if rest != "" {
						parts = strings.Split(rest, sep)
					}
				}
			}
		} else {
			parts = strings.Split(path, sep)
		}
	} else {
		// Unix: check for leading /
		if strings.HasPrefix(path, "/") {
			prefix = "/"
			rest := strings.TrimPrefix(path, "/")
			if rest != "" {
				parts = strings.Split(rest, sep)
			}
		} else {
			parts = strings.Split(path, sep)
		}
	}

	// Process . and ..
	var out []string
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(out) == 0 {
				// Trying to .. above the prefix/root
				if prefix != "" {
					// Stop at root, don't escape
					continue
				}
				// No prefix - allow relative .. paths
				out = append(out, part)
			} else if out[len(out)-1] == ".." {
				// Previous element is also .., can't collapse
				out = append(out, part)
			} else {
				// Previous element is a normal directory, remove it
				out = out[:len(out)-1]
			}
		} else {
			out = append(out, part)
		}
	}

	// Build result
	var result string
	if prefix != "" {
		if len(out) == 0 {
			result = prefix
		} else {
			result = prefix + strings.Join(out, sep)
		}
	} else {
		if len(out) == 0 {
			return "."
		}
		result = strings.Join(out, sep)
	}

	// Restore local prefix, or add one to preserve the detected style.
	// Every relative result gets a "./" (or ".\") prefix so the path is
	// self-describing: Split and Join can recover the style without
	// external context.
	if localPrefix != "" {
		result = localPrefix + result
	} else if prefix == "" && result != "." {
		result = "." + sep + result
	}

	// Restore trailing separator
	if hadTrailing && !strings.HasSuffix(result, sep) {
		result += sep
	}

	return result
}

func isDriveLetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

// pathStyle represents the detected path style
type pathStyle struct {
	kind styleKind
	sep  rune
}

type styleKind int

const (
	styleUnix styleKind = iota
	styleWindows
	styleURL
)

// detectStyle determines the path style from the elements.
// Drive letters and URL protocols are checked first. Otherwise, the first
// separator character (/ or \) encountered across all elements determines
// the style: forward slash means Unix, backslash means Windows.
func detectStyle(elem []string) pathStyle {
	for _, e := range elem {
		if e == "" {
			continue
		}

		// Check for Windows style: [letter]:
		if len(e) >= 2 && e[1] == ':' && isDriveLetter(e[0]) {
			return pathStyle{kind: styleWindows, sep: '\\'}
		}

		// Check for URL style: protocol:// (requires non-empty protocol)
		if idx := strings.Index(e, "://"); idx > 0 {
			return pathStyle{kind: styleURL, sep: '/'}
		}

		// First separator character determines style
		for i := 0; i < len(e); i++ {
			if e[i] == '/' {
				return pathStyle{kind: styleUnix, sep: '/'}
			}
			if e[i] == '\\' {
				return pathStyle{kind: styleWindows, sep: '\\'}
			}
		}
	}

	// Default to Unix
	return pathStyle{kind: styleUnix, sep: '/'}
}

// isRoot checks if a path is a root for the given style
func isRoot(path string, style pathStyle) bool {
	switch style.kind {
	case styleUnix:
		return path == "/"
	case styleWindows:
		return len(path) == 3 &&
			isDriveLetter(path[0]) &&
			path[1] == ':' && path[2] == '\\'
	case styleURL:
		_, after, ok := strings.Cut(path, "://")
		if !ok {
			return false
		}
		slashCount := strings.Count(after, "/")
		if slashCount == 0 {
			return true
		}
		return slashCount == 1 && strings.HasSuffix(path, "/")
	}
	return false
}

func splitAll(path string) []string {
	if path == "" {
		return nil
	}
	var result []string
	for path != "" {
		dir, file := Split(path)
		if file != "" {
			result = append([]string{file}, result...)
		}
		if dir == path {
			if dir != "" {
				result = append([]string{dir}, result...)
			}
			break
		}
		path = dir
	}
	return result
}

// joinParts joins path parts according to the style.
func joinParts(parts []string, style pathStyle) string {
	if len(parts) == 0 {
		return ""
	}

	// Trim leading empty strings (prevent unwanted leading separators)
	var start int
	for start < len(parts) && parts[start] == "" {
		start++
	}
	if start >= len(parts) {
		return ""
	}
	parts = parts[start:]

	sep := string(style.sep)

	switch style.kind {
	case styleURL:
		// For URLs, first part might be proto:// or proto://host/ (a root)
		if len(parts) > 0 && strings.Contains(parts[0], "://") {
			if len(parts) == 1 {
				return parts[0]
			}
			// If first part is a root (ends with /), don't add extra sep
			first := parts[0]
			if strings.HasSuffix(first, "/") {
				return first + strings.Join(parts[1:], sep)
			}
			// Otherwise join normally
			return strings.Join(parts, sep)
		}
		return strings.Join(parts, sep)

	case styleWindows:
		// For Windows, first part might be C: or C:\
		if len(parts) > 0 && len(parts[0]) >= 2 &&
			parts[0][1] == ':' &&
			isDriveLetter(parts[0][0]) {
			first := parts[0]
			if len(parts) == 1 {
				// Single drive letter - ensure it has backslash
				if !strings.HasSuffix(first, sep) {
					return first + sep
				}
				return first
			}
			// Multiple parts - if first is a root (C:\), don't add extra sep
			if strings.HasSuffix(first, sep) {
				return first + strings.Join(parts[1:], sep)
			}
			// Otherwise add separator
			return first + sep + strings.Join(parts[1:], sep)
		}
		return strings.Join(parts, sep)

	case styleUnix:
		// For Unix, check if first part is root /
		if len(parts) > 0 && parts[0] == "/" {
			if len(parts) == 1 {
				return "/"
			}
			return "/" + strings.Join(parts[1:], sep)
		}
		return strings.Join(parts, sep)

	default:
		return strings.Join(parts, sep)
	}
}

// Match reports whether name matches the shell pattern.
// The pattern syntax is the same as in path.Match from the standard library.
// This is an alias to avoid importing both packages.
func Match(pattern, name string) (matched bool, err error) {
	return stdpath.Match(pattern, name)
}

// ErrBadPattern indicates a pattern was malformed.
// This is an alias to avoid importing both packages.
var ErrBadPattern = stdpath.ErrBadPattern
