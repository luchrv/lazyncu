package detect

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
)

// maxNodeSearchDepth bounds how many levels below the root a package.json
// is searched for: 1 covers the root project itself, 2 a folder of
// projects, 3 a monorepo's packages/*/package.json layout.
const maxNodeSearchDepth = 3

// errNodeProjectFound stops the walk early once a package.json is seen.
var errNodeProjectFound = errors.New("node project found")

// HasNodeProject reports whether dir holds a package.json at its root or in
// any subdirectory within maxNodeSearchDepth levels, skipping node_modules
// and hidden directories. A missing or unreadable dir reports false.
func HasNodeProject(dir string) bool {
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable subtree: keep walking the rest
		}
		if d.IsDir() {
			return dirAction(dir, path, d.Name())
		}
		if d.Name() == "package.json" {
			return errNodeProjectFound
		}
		return nil
	})
	return errors.Is(err, errNodeProjectFound)
}

// dirAction decides whether the walk descends into a directory: the root
// always, others only when visible, not node_modules, and shallow enough
// for their files to sit within the search depth.
func dirAction(root, path, name string) error {
	if path == root {
		return nil
	}
	if name == "node_modules" || strings.HasPrefix(name, ".") {
		return fs.SkipDir
	}
	if depthBelow(root, path) >= maxNodeSearchDepth {
		return fs.SkipDir
	}
	return nil
}

// depthBelow counts how many levels path sits below root (root itself is 0).
func depthBelow(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}
