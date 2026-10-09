// Package scanner resolves pending upgrades for projects and the global
// package set. Registry metadata comes through an injected Registry (production:
// `npm view`); `npm ls -g` still runs through the injected Runner. Results are
// immutable.
package scanner

import (
	"context"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/registry"
	"github.com/luchrv/lazyncu/semver"
)

// Runner executes an external command in dir and returns its stdout. It is
// injected so tests can supply canned output without spawning processes.
// Implementations must honor ctx cancellation. A non-nil error may still be
// accompanied by usable stdout (e.g. npm exits non-zero with valid JSON).
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) (stdout []byte, err error)
}

// Registry retrieves a package's registry metadata. The production
// implementation queries the registry npm's configuration selects for the
// project (including private and scoped registries and their credentials),
// falling back to `npm view`.
type Registry interface {
	// Fetch resolves pkg with the npm configuration in effect in dir.
	Fetch(ctx context.Context, dir, pkg string) (registry.Metadata, error)
	// PublishTimes is consulted only when a release cooldown is configured.
	PublishTimes(ctx context.Context, dir, pkg string) (map[string]string, error)
	// MinReleaseAge returns npm's raw min-release-age setting for dir ("" when
	// unset).
	MinReleaseAge(ctx context.Context, dir string) (string, error)
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
	// Nvmrc is the trimmed .nvmrc content; empty when the file is absent.
	Nvmrc string
	// EnginesNode is the engines.node constraint from package.json; empty when undeclared.
	EnginesNode string
	// Warnings surfaces .ncurc options or configs outside the supported subset,
	// so the UI can tell the user this project's results may differ from ncu.
	Warnings []string
}
