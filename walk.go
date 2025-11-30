package fs

import (
	"cmp"
	"context"
	"iter"
	"slices"

	"lesiw.io/fs/path"
)

// A ReadDirFS is a file system with the ReadDir method.
//
// If not implemented, ReadDir falls back to [Walk] with depth 1.
type ReadDirFS interface {
	FS

	// ReadDir reads the directory and returns an iterator over its entries.
	// For entries from ReadDir, Path() returns empty string.
	ReadDir(ctx context.Context, name string) iter.Seq2[DirEntry, error]
}

// A WalkFS is a file system with the Walk method.
//
// If not implemented, Walk falls back to breadth-first traversal using
// [ReadDirFS].
type WalkFS interface {
	FS

	// Walk traverses the filesystem starting at root.
	// The depth parameter controls how deep to traverse:
	//   depth <= 0: unlimited depth (like find without -maxdepth)
	//   depth >= 1: root directory plus n-1 levels of subdirectories
	//               (like find -maxdepth n)
	//
	// Entries returned by Walk have Path() populated with full paths.
	Walk(
		ctx context.Context, root string, depth int,
	) iter.Seq2[DirEntry, error]
}

// ReadDir reads the named directory and returns an iterator over its
// entries. Analogous to: [os.ReadDir], [io/fs.ReadDir], ls, 9P Tread on
// directory.
//
// Requires: [ReadDirFS] || [WalkFS]
func ReadDir(
	ctx context.Context, fsys FS, name string,
) iter.Seq2[DirEntry, error] {
	if rdfs, ok := fsys.(ReadDirFS); ok {
		return rdfs.ReadDir(ctx, name)
	}

	// Fallback to Walk if available
	if wfs, ok := fsys.(WalkFS); ok {
		return wfs.Walk(ctx, name, 1)
	}

	// No ReadDir or Walk support
	return func(yield func(DirEntry, error) bool) {
		yield(nil, &PathError{
			Op:   "readdir",
			Path: name,
			Err:  ErrUnsupported,
		})
	}
}

// Walk traverses the filesystem rooted at root.
// Analogous to: [io/fs.WalkDir], find, tree.
//
// The depth parameter controls how deep to traverse (like find -maxdepth):
//   - depth <= 0: unlimited depth (like find without -maxdepth)
//   - depth >= 1: root directory plus n-1 levels of subdirectories
//     (like find -maxdepth n)
//
// Walk does not guarantee any particular order (lexicographic or
// breadth-first). Implementations may choose whatever order is most
// efficient. For guaranteed lexicographic order within each directory,
// use [ReadDir].
//
// Walk does not follow symbolic links. Entries are yielded for symbolic
// links themselves, but they are not traversed.
//
// Entries returned by Walk have Path() populated with the full path.
//
// If an error occurs reading a directory, the iteration yields a zero
// DirEntry and the error. The caller can choose to continue iterating
// (skip that directory) or break to stop the walk.
//
// Requires: [WalkFS] || [ReadDirFS]
func Walk(
	ctx context.Context, fsys FS, root string, depth int,
) iter.Seq2[DirEntry, error] {
	if wfs, ok := fsys.(WalkFS); ok {
		return wfs.Walk(ctx, root, depth)
	}

	// Fallback to ReadDir if available
	if _, ok := fsys.(ReadDirFS); ok {
		return walkBreadthFirst(ctx, fsys, root, depth)
	}

	// No Walk or ReadDir support
	return func(yield func(DirEntry, error) bool) {
		yield(nil, &PathError{
			Op:   "walk",
			Path: root,
			Err:  ErrUnsupported,
		})
	}
}

// readDirEntry implements DirEntry for ReadDir (no path/depth).
type readDirEntry struct {
	name  string
	isDir bool
	typ   Mode
	info  FileInfo
}

func (de *readDirEntry) Name() string            { return de.name }
func (de *readDirEntry) IsDir() bool             { return de.isDir }
func (de *readDirEntry) Type() Mode              { return de.typ }
func (de *readDirEntry) Info() (FileInfo, error) { return de.info, nil }
func (de *readDirEntry) Path() string            { return "" }

// walkEntry implements DirEntry with path information for Walk.
type walkEntry struct {
	name  string
	isDir bool
	typ   Mode
	info  FileInfo
	path  string
}

func (we *walkEntry) Name() string            { return we.name }
func (we *walkEntry) IsDir() bool             { return we.isDir }
func (we *walkEntry) Type() Mode              { return we.typ }
func (we *walkEntry) Info() (FileInfo, error) { return we.info, nil }
func (we *walkEntry) Path() string            { return we.path }

// queueItem represents a directory to be processed
type queueItem struct {
	path  string
	depth int
}

// walkBreadthFirst implements breadth-first traversal using ReadDirFS.
func walkBreadthFirst(
	ctx context.Context, fsys FS, root string, depth int,
) iter.Seq2[DirEntry, error] {
	return func(yield func(DirEntry, error) bool) {
		// Start with root directory
		queue := []queueItem{{root, 0}}

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			// Read directory entries
			var entries []DirEntry
			for entry, err := range ReadDir(ctx, fsys, current.path) {
				if err != nil {
					// Yield error for this directory and continue
					if !yield(nil, &PathError{
						Op:   "readdir",
						Path: current.path,
						Err:  err,
					}) {
						return
					}
					break
				}
				entries = append(entries, entry)
			}

			// Sort entries lexicographically
			slices.SortFunc(entries, func(a, b DirEntry) int {
				return cmp.Compare(a.Name(), b.Name())
			})

			// Process entries at this level
			for _, entry := range entries {
				// Build full path for this entry
				entryPath := path.Join(current.path, entry.Name())

				// Get FileInfo for the entry
				info, err := entry.Info()
				if err != nil {
					if !yield(nil, &PathError{
						Op:   "stat",
						Path: entryPath,
						Err:  err,
					}) {
						return
					}
					continue
				}

				// Wrap entry with path
				we := &walkEntry{
					name:  entry.Name(),
					isDir: entry.IsDir(),
					typ:   entry.Type(),
					info:  info,
					path:  entryPath,
				}

				// Yield wrapped entry
				if !yield(we, nil) {
					return
				}

				// Queue subdirectories for next level if within depth
				// depth <= 0 means unlimited
				// depth = 1 means only immediate children (no subdirs)
				// depth = 2 means immediate children + 1 level of subdirs,
				// etc.
				if entry.IsDir() {
					nextDepth := current.depth + 1
					if depth <= 0 || nextDepth < depth {
						queue = append(queue, queueItem{
							path:  entryPath,
							depth: nextDepth,
						})
					}
				}
			}
		}
	}
}
