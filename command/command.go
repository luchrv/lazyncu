// Package command builds the copyable, never-executed update commands the
// dashboard suggests for each context (read-only rule).
package command

import (
	"strings"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/scanner"
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

// ProjectUpdate builds `cd <dir> && <add> <pkg>@<ver> …` for the given
// packages, where <add> matches the project's package manager. No packages
// yields an empty string.
func ProjectUpdate(dir string, pm detect.PackageManager, pkgs []scanner.Package) string {
	if len(pkgs) == 0 {
		return ""
	}
	specs := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		specs = append(specs, p.Name+"@"+p.New)
	}
	return "cd " + dir + " && " + addStep(pm) + " " + strings.Join(specs, " ")
}

// addStep is the package-manager add verb used to install resolved versions.
func addStep(pm detect.PackageManager) string {
	switch pm {
	case detect.Pnpm:
		return "pnpm add"
	case detect.Yarn:
		return "yarn add"
	default:
		return "npm install"
	}
}
