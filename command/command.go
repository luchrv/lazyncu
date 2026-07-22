// Package command builds the copyable, never-executed update commands the
// dashboard suggests for each context (read-only rule).
package command

import (
	"strings"

	"github.com/luchrv/ncu-tui/detect"
	"github.com/luchrv/ncu-tui/scanner"
)

// GlobalUpdate builds `npm install -g pkg@ver ...` from global scan results.
// No upgradable packages yields an empty string (no command to show).
func GlobalUpdate(pkgs []scanner.Package) string {
	if len(pkgs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(pkgs)+3)
	parts = append(parts, "npm", "install", "-g")
	for _, p := range pkgs {
		parts = append(parts, p.Name+"@"+p.New)
	}
	return strings.Join(parts, " ")
}

// ProjectUpdate builds `cd <dir> && ncu -u && <install>` with the install
// step matching the project's package manager.
func ProjectUpdate(dir string, pm detect.PackageManager) string {
	install := map[detect.PackageManager]string{
		detect.Npm:  "npm install",
		detect.Pnpm: "pnpm install",
		detect.Yarn: "yarn",
	}[pm]
	if install == "" {
		install = "npm install"
	}
	return "cd " + dir + " && ncu -u && " + install
}
