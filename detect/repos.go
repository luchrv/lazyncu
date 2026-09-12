package detect

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Repo is one repository discovered inside a folder-mode path.
type Repo struct {
	// Dir is the repository directory as reached from the folder: a path
	// that goes through a symbolic link keeps the link, so commands run
	// where the user sees the entry.
	Dir string
	// Label is Dir relative to the folder ("group/api", or the link's name).
	Label string
}

// Repos discovers the repositories under root: the first directory on each
// branch that holds a package.json, at any depth. Symbolic links to
// directories are followed; each real directory is entered at most once
// (deduplicating repositories reachable twice and cutting link cycles);
// node_modules and dot-prefixed directories are skipped; unreadable entries
// and dangling links are skipped. Only an unreadable root is an error.
// The result is in deterministic, name-sorted depth-first order.
func Repos(root string) ([]Repo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading folder %s: %w", root, err)
	}
	realRoot := realPath(root)
	w := &repoWalker{root: root, realRoot: realRoot, visited: map[string]bool{realRoot: true}}
	w.visitEntries(root, entries)
	return w.repos, nil
}

// repoWalker carries the discovery state of one Repos call.
type repoWalker struct {
	root     string
	realRoot string
	visited  map[string]bool
	repos    []Repo
}

func (w *repoWalker) visitEntries(dir string, entries []fs.DirEntry) {
	for _, entry := range entries {
		if isIgnoredDir(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if resolvesToDir(entry, path) {
			w.visitDir(path)
		}
	}
}

// visitDir records path as a repository when it holds a manifest, otherwise
// descends into it — unless its real directory was already entered or is
// the folder itself / one of its ancestors (a link pointing back up).
func (w *repoWalker) visitDir(path string) {
	key := realPath(path)
	if w.visited[key] || isAncestorOrSelf(key, w.realRoot) {
		return
	}
	w.visited[key] = true
	if hasManifest(path) {
		w.repos = append(w.repos, Repo{Dir: path, Label: repoLabel(w.root, path)})
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return // unreadable subtree: skip it, keep the rest
	}
	w.visitEntries(path, entries)
}

// resolvesToDir reports whether entry is a directory, following a symbolic
// link to find out; a dangling link resolves to nothing.
func resolvesToDir(entry fs.DirEntry, path string) bool {
	if entry.IsDir() {
		return true
	}
	if entry.Type()&fs.ModeSymlink == 0 {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasManifest(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "package.json"))
	return err == nil && !info.IsDir()
}

// realPath is the symlink-resolved, cleaned form used as the visited key;
// it falls back to the cleaned path when resolution fails.
func realPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

// isAncestorOrSelf reports whether dir equals root or contains it, compared
// per path segment on already-resolved paths.
func isAncestorOrSelf(dir, root string) bool {
	return dir == root || strings.HasPrefix(root, dir+string(filepath.Separator))
}

func repoLabel(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return dir
	}
	return rel
}
