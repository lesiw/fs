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
// (Unix, Windows, or URL) from the input and applying appropriate rules.
// Empty elements are ignored, except if the last element is empty, which
// adds a trailing separator to indicate a directory.
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

	// Detect path style from first non-empty element
	style := detectStyle(elem)

	// Filter empty elements (but remember if last was empty for trailing sep)
	var trailingDir bool
	if len(elem) > 0 && elem[len(elem)-1] == "" {
		trailingDir = true
		elem = elem[:len(elem)-1]
	}

	var parts []string
	for _, e := range elem {
		if e != "" {
			parts = append(parts, e)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	result := joinParts(parts, style)

	if trailingDir && !strings.HasSuffix(result, string(style.sep)) {
		result += string(style.sep)
	}

	return Clean(result)
}

// Split splits path into directory and file components.
// The directory does not include a trailing separator, except for roots.
// Returns ("", file) if path has no directory component.
// Returns (dir, "") if path ends with a trailing separator (is a directory).
func Split(path string) (dir, file string) {
	style := detectStyle([]string{path})
	sep := string(style.sep)

	// Handle trailing separator (directory)
	if strings.HasSuffix(path, sep) {
		if isRoot(path, style) {
			// Root directory: return (root, "")
			return path, ""
		}
		// Non-root directory: remove trailing sep and return as dir
		path = strings.TrimSuffix(path, sep)
		dir = path
		file = ""
		return
	}

	// Find last separator
	i := strings.LastIndex(path, sep)
	if i < 0 {
		// No separator - entire path is the file
		return "", path
	}

	dir = path[:i+1] // Include the separator temporarily
	file = path[i+1:]

	// Check if dir (with separator) is a root
	if isRoot(dir, style) {
		return dir, file
	}

	// Not a root, remove the trailing separator
	dir = strings.TrimSuffix(dir, sep)

	return dir, file
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
//   - Paths starting with [letter]:\ or [letter]:/ (Windows-style)
//   - Paths starting with [protocol]:// (URL-style)
func IsAbs(path string) bool {
	if path == "" {
		return false
	}

	// Unix-style: starts with /
	if path[0] == '/' {
		return true
	}

	// Windows-style: starts with [letter]:\
	if len(path) >= 3 && path[1] == ':' &&
		(path[2] == '\\' || path[2] == '/') &&
		((path[0] >= 'A' && path[0] <= 'Z') ||
			(path[0] >= 'a' && path[0] <= 'z')) {
		return true
	}

	// URL-style: contains ://
	if strings.Contains(path, "://") {
		return true
	}

	return false
}

// Clean returns the shortest path name equivalent to path by purely lexical
// processing. It applies the following rules iteratively until no further
// processing can be done:
//
//  1. Replace multiple separators with a single separator
//  2. Eliminate redundant . path elements
//  3. Eliminate .. path elements and the preceding element
//  4. Preserve trailing separator (indicates directory)
//  5. Preserve leading ./ or .\ (indicates local path style)
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
		path = path[2:]
	}

	// Split into parts, preserving special prefixes
	var prefix string
	var parts []string

	if style.kind == styleURL {
		// For URLs, extract protocol://host as a single "root" part
		protoEnd := strings.Index(path, "://")
		if protoEnd >= 0 {
			// Find the first / after ://
			hostStart := protoEnd + 3
			hostEnd := strings.Index(path[hostStart:], "/")
			if hostEnd < 0 {
				// No path after host, just protocol://host
				prefix = path
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
		// For Windows, preserve drive letter
		if len(path) >= 2 && path[1] == ':' {
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
			} else {
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

	// Restore local prefix if it was present
	if localPrefix != "" {
		result = localPrefix + result
	}

	// Restore trailing separator
	if hadTrailing && !strings.HasSuffix(result, sep) {
		result += sep
	}

	return result
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

// detectStyle determines the path style from the elements
func detectStyle(elem []string) pathStyle {
	for _, e := range elem {
		if e == "" {
			continue
		}

		// Check for Windows style: [letter]:
		if len(e) >= 2 && e[1] == ':' &&
			((e[0] >= 'A' && e[0] <= 'Z') || (e[0] >= 'a' && e[0] <= 'z')) {
			return pathStyle{kind: styleWindows, sep: '\\'}
		}

		// Check for URL style: protocol://
		if strings.Contains(e, "://") {
			return pathStyle{kind: styleURL, sep: '/'}
		}

		// Check for existing backslashes (Windows-style path)
		if strings.Contains(e, "\\") {
			return pathStyle{kind: styleWindows, sep: '\\'}
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
		// C:\, D:\, etc.
		return len(path) == 3 && path[1] == ':' && path[2] == '\\'
	case styleURL:
		// Must end with / and have ://
		if !strings.HasSuffix(path, "/") {
			return false
		}
		protoEnd := strings.Index(path, "://")
		if protoEnd < 0 {
			return false
		}
		// Root is protocol://host/
		rest := path[protoEnd+3:]
		return strings.Count(rest, "/") == 1
	}
	return false
}

// joinParts joins path parts according to the style
func joinParts(parts []string, style pathStyle) string {
	if len(parts) == 0 {
		return ""
	}

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
		if len(parts) > 0 && len(parts[0]) >= 2 && parts[0][1] == ':' {
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
