package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/ncurc"
	"github.com/luchrv/lazyncu/registry"
	"github.com/luchrv/lazyncu/semver"
)

// Scanner resolves pending upgrades for projects and the global package set.
type Scanner struct {
	runner Runner
	reg    Registry
}

// New builds a Scanner around a command Runner (for `npm ls -g`) and a
// registry client (for upgrade targets and npm settings).
func New(runner Runner, reg Registry) Scanner {
	return Scanner{runner: runner, reg: reg}
}

// ScanGlobal checks globally installed packages: `npm ls -g` provides the
// package list and current versions, and the registry provides targets. A
// failed `npm ls -g` degrades to no packages rather than failing the source.
func (s Scanner) ScanGlobal(ctx context.Context) ([]Package, error) {
	installed := s.globalInstalledVersions(ctx)
	if installed == nil {
		installed = map[string]string{}
	}
	return s.resolve(ctx, installed, resolveOptions{cooldown: s.cooldown(ctx, "")}), nil
}

// ErrFolderNotScannable reports a direct scan of a folder of repositories:
// folders are expanded by the orchestrator into one ScanPath per repository.
var ErrFolderNotScannable = errors.New("folder of repositories must be expanded into its repositories before scanning")

// ScanPath scans one path — a single project or a workspaces monorepo —
// choosing the strategy via detect. A folder of repositories is refused:
// callers discover its repositories and scan each one.
func (s Scanner) ScanPath(ctx context.Context, dir string) ([]Project, error) {
	switch detect.ScanMode(dir) {
	case detect.ModeDeep:
		return s.scanDeep(ctx, dir)
	case detect.ModeFolder:
		return nil, fmt.Errorf("%s: %w", dir, ErrFolderNotScannable)
	default:
		return s.scanSingle(ctx, dir)
	}
}

func (s Scanner) scanSingle(ctx context.Context, dir string) ([]Project, error) {
	cooldown := s.cooldown(ctx, dir)
	return []Project{s.scanManifest(ctx, dir, ".", cooldown)}, nil
}

func (s Scanner) scanDeep(ctx context.Context, root string) ([]Project, error) {
	manifests, err := detect.Manifests(root)
	if err != nil {
		return nil, err
	}
	cooldown := s.cooldown(ctx, root)
	projects := make([]Project, 0, len(manifests))
	for _, m := range manifests {
		projects = append(projects, s.scanManifest(ctx, m.Dir, m.Label, cooldown))
	}
	return projects, nil
}

// scanManifest reads one manifest's dependencies and resolves each registry
// spec against the registry, applying the project's engines, .ncurc options
// and the npm cooldown.
func (s Scanner) scanManifest(ctx context.Context, dir, label string, cooldown int) Project {
	deps := manifestVersions(filepath.Join(dir, "package.json"))
	nvmrc, engines := detect.NodeContext(dir)
	cfg := ncurc.Load(dir)
	opts := resolveOptions{dir: dir, enginesNode: engines, ncurc: cfg.Options, cooldown: cooldown}
	packages := s.resolve(ctx, deps, opts)
	return Project{
		Dir:         dir,
		Label:       label,
		PM:          detect.PackageManagerFor(dir),
		Packages:    packages,
		Counters:    countSeverities(packages),
		Nvmrc:       nvmrc,
		EnginesNode: engines,
		Warnings:    cfg.Warnings,
	}
}

// resolveOptions carries the per-project settings that shape target selection.
type resolveOptions struct {
	// dir selects the npm configuration for registry lookups ("" = global).
	dir         string
	enginesNode string
	ncurc       ncurc.Options
	cooldown    int
}

// allows applies the .ncurc filter/reject rules to a dependency and its
// declared spec.
func (o resolveOptions) allows(name, current string) bool {
	if len(o.ncurc.Filter) > 0 && !ncurc.MatchesAny(o.ncurc.Filter, name) {
		return false
	}
	if ncurc.MatchesAny(o.ncurc.Reject, name) {
		return false
	}
	if len(o.ncurc.FilterVersion) > 0 && !ncurc.MatchesAny(o.ncurc.FilterVersion, current) {
		return false
	}
	if ncurc.MatchesAny(o.ncurc.RejectVersion, current) {
		return false
	}
	return true
}

// registryOptions builds the registry selection options for one dependency.
func (o resolveOptions) registryOptions(current string) registry.Options {
	opts := registry.Options{
		EnginesNode:     o.enginesNode,
		AllowDeprecated: o.ncurc.AllowDeprecated(),
		AllowPre:        o.ncurc.Pre || registry.IsPre(current),
		CooldownDays:    o.cooldown,
	}
	if tag, ok := o.ncurc.DistTag(); ok {
		opts.DistTag = tag
	}
	return opts
}

// resolve fetches metadata for every registry spec concurrently and returns
// the upgradable packages, sorted by name. A dependency whose fetch fails is
// omitted without failing the scan.
func (s Scanner) resolve(ctx context.Context, deps map[string]string, opts resolveOptions) []Package {
	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]*Package, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		current := deps[name]
		if !registry.IsRegistrySpec(current) || !opts.allows(name, current) {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			meta, err := s.fetch(ctx, opts.dir, name, opts.cooldown > 0)
			if err != nil {
				return
			}
			target, ok := registry.Target(meta, current, opts.registryOptions(current))
			if !ok {
				return
			}
			results[i] = &Package{
				Name:     name,
				Current:  current,
				New:      target,
				Severity: semver.Classify(current, target),
			}
		}()
	}
	wg.Wait()

	packages := make([]Package, 0, len(names))
	for _, p := range results {
		if p != nil {
			packages = append(packages, *p)
		}
	}
	return packages
}

// fetch retrieves a package's metadata and, when a cooldown applies, attaches
// the publish time of every version that lacks one.
func (s Scanner) fetch(ctx context.Context, dir, name string, withTimes bool) (registry.Metadata, error) {
	meta, err := s.reg.Fetch(ctx, dir, name)
	if err != nil || !withTimes {
		return meta, err
	}
	times, err := s.reg.PublishTimes(ctx, dir, name)
	if err != nil {
		return registry.Metadata{}, err
	}
	for i := range meta.Versions {
		if meta.Versions[i].Time == "" {
			meta.Versions[i].Time = times[meta.Versions[i].Version]
		}
	}
	return meta, nil
}

// cooldown reads npm's min-release-age setting (in days) for a directory, so
// versions published more recently are excluded. A missing or invalid setting,
// or a failed lookup, means no cooldown.
func (s Scanner) cooldown(ctx context.Context, dir string) int {
	raw, err := s.reg.MinReleaseAge(ctx, dir)
	if err != nil {
		return 0
	}
	return parseCooldown(strings.TrimSpace(raw))
}

// parseCooldown accepts a day count ("7", "7d") or a duration with an h/m
// suffix, rounded down to whole days; anything else yields 0.
func parseCooldown(raw string) int {
	if raw == "" || raw == "undefined" || raw == "null" {
		return 0
	}
	suffix := raw[len(raw)-1]
	if suffix >= '0' && suffix <= '9' {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
		return 0
	}
	n, err := strconv.Atoi(raw[:len(raw)-1])
	if err != nil || n <= 0 {
		return 0
	}
	switch suffix {
	case 'd':
		return n
	case 'h':
		return n / 24
	case 'm':
		return n / (24 * 60)
	default:
		return 0
	}
}

// globalInstalledVersions returns installed global versions, tolerating a
// non-zero npm exit as long as stdout holds valid JSON (npm ls does that when
// it finds extraneous packages). Any hard failure yields nil.
func (s Scanner) globalInstalledVersions(ctx context.Context) map[string]string {
	out, err := s.runner.Run(ctx, "", "npm", "ls", "-g", "--depth=0", "--json")
	if err != nil && len(out) == 0 {
		return nil
	}
	var report struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if json.Unmarshal(out, &report) != nil {
		return nil
	}
	versions := make(map[string]string, len(report.Dependencies))
	for name, dep := range report.Dependencies {
		versions[name] = dep.Version
	}
	return versions
}

func countSeverities(packages []Package) semver.Counters {
	severities := make([]semver.Severity, len(packages))
	for i, p := range packages {
		severities[i] = p.Severity
	}
	return semver.Count(severities)
}

// manifestVersions reads dependency versions declared in a package.json,
// including the packageManager field npm-check-updates treats as a dependency.
// A missing or malformed manifest yields an empty map, never an error.
func manifestVersions(pkgFile string) map[string]string {
	data, err := os.ReadFile(pkgFile)
	if err != nil {
		return nil
	}
	var manifest struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
		PackageManager       string            `json:"packageManager"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return nil
	}
	versions := make(map[string]string,
		len(manifest.Dependencies)+len(manifest.DevDependencies)+len(manifest.OptionalDependencies)+1)
	for _, group := range []map[string]string{
		manifest.Dependencies, manifest.DevDependencies, manifest.OptionalDependencies,
	} {
		maps.Copy(versions, group)
	}
	if name, spec, ok := splitPackageManager(manifest.PackageManager); ok {
		versions[name] = spec
	}
	return versions
}

// splitPackageManager parses a packageManager field ("pnpm@9.0.0") into its
// name and version spec.
func splitPackageManager(field string) (name, spec string, ok bool) {
	idx := strings.LastIndex(field, "@")
	if idx <= 0 || idx == len(field)-1 {
		return "", "", false
	}
	return field[:idx], field[idx+1:], true
}
