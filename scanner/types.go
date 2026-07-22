// Package scanner executes ncu and npm (through an injected Runner) and
// parses their JSON output into immutable scan results.
package scanner

import (
	"context"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/semver"
)

// Runner executes an external command in dir and returns its stdout. It is
// injected so tests can supply canned output without spawning processes.
// Implementations must honor ctx cancellation. A non-nil error may still be
// accompanied by usable stdout (e.g. npm exits non-zero with valid JSON).
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) (stdout []byte, err error)
}

// Package is one upgradable dependency.
type Package struct {
	Name     string
	Current  string
	New      string
	Severity semver.Severity
}

// Project is one discovered package.json with its pending upgrades.
type Project struct {
	// Dir is the absolute directory containing the package.json.
	Dir string
	// Label is the path relative to the registered source root ("." for the root itself).
	Label    string
	PM       detect.PackageManager
	Packages []Package
	Counters semver.Counters
}
