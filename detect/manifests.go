package detect

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
)

// Manifest is one discovered package.json in a deep scan.
type Manifest struct {
	// Dir is the directory holding the package.json.
	Dir string
	// Label is Dir relative to the scanned root ("." for the root itself).
	Label string
}

// Manifests discovers every package.json under root recursively — the root
// manifest plus nested manifests — skipping node_modules and .pnpm-store,
// matching npm-check-updates' `--deep` (`--packageFile '**/package.json'`).
// Results are sorted by label for a deterministic order. Only an unreadable
// root is an error; unreadable subtrees are skipped.
func Manifests(root string) ([]Manifest, error) {
	var out []Manifest
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return fmt.Errorf("reading folder %s: %w", root, err)
			}
			return nil
		}
		if d.IsDir() {
			if path != root && isDeepIgnoredDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() != "package.json" {
			return nil
		}
		dir := filepath.Dir(path)
		out = append(out, Manifest{Dir: dir, Label: manifestLabel(root, dir)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out, nil
}

// isDeepIgnoredDir reports directories a deep walk never enters: dependency
// trees and the pnpm content-addressable store.
func isDeepIgnoredDir(name string) bool {
	return name == "node_modules" || name == ".pnpm-store"
}

// manifestLabel is dir relative to root ("." for the root itself).
func manifestLabel(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return dir
	}
	return rel
}
