// Package detect infers, per registered path, the ncu scan mode (single
// project vs. deep) and the project's package manager from its lockfile.
// Detection is stateless: it re-reads the filesystem on every call.
package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Mode is the ncu invocation strategy for a path.
type Mode string

const (
	// ModeSingle scans one package.json with plain `ncu`.
	ModeSingle Mode = "single"
	// ModeDeep scans recursively with `ncu --deep` (folder of projects or monorepo).
	ModeDeep Mode = "deep"
)

// PackageManager identifies the tool that owns a project's lockfile.
type PackageManager string

const (
	Npm  PackageManager = "npm"
	Pnpm PackageManager = "pnpm"
	Yarn PackageManager = "yarn"
)

// ScanMode applies the decision tree: no package.json → deep (folder of
// projects); package.json with a workspaces field or pnpm-workspace.yaml
// present → deep (monorepo); otherwise single.
func ScanMode(dir string) Mode {
	pkg, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return ModeDeep
	}
	if _, err := os.Stat(filepath.Join(dir, "pnpm-workspace.yaml")); err == nil {
		return ModeDeep
	}
	if hasWorkspacesField(pkg) {
		return ModeDeep
	}
	return ModeSingle
}

// PackageManagerFor detects a project's package manager from its lockfile,
// with precedence pnpm > yarn > npm and npm as the default.
func PackageManagerFor(dir string) PackageManager {
	ordered := []struct {
		lockfile string
		pm       PackageManager
	}{
		{"pnpm-lock.yaml", Pnpm},
		{"yarn.lock", Yarn},
		{"package-lock.json", Npm},
	}
	for _, candidate := range ordered {
		if _, err := os.Stat(filepath.Join(dir, candidate.lockfile)); err == nil {
			return candidate.pm
		}
	}
	return Npm
}

func hasWorkspacesField(packageJSON []byte) bool {
	var manifest struct {
		Workspaces json.RawMessage `json:"workspaces"`
	}
	if err := json.Unmarshal(packageJSON, &manifest); err != nil {
		return false
	}
	return len(manifest.Workspaces) > 0
}
