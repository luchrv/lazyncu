package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/luchrv/ncu-tui/detect"
	"github.com/luchrv/ncu-tui/semver"
)

// MinNcuMajor is the minimum supported major version of npm-check-updates.
const MinNcuMajor = 18

// Scanner runs ncu/npm through an injected Runner and parses their output.
type Scanner struct {
	runner Runner
}

// New builds a Scanner around the given Runner.
func New(runner Runner) Scanner {
	return Scanner{runner: runner}
}

// Preflight verifies that ncu is on PATH with a supported version.
func (s Scanner) Preflight(ctx context.Context) error {
	out, err := s.runner.Run(ctx, "", "ncu", "--version")
	if err != nil {
		return fmt.Errorf(
			"ncu is not available: %w\ninstall it with: npm install -g npm-check-updates", err)
	}
	version := strings.TrimSpace(string(out))
	major, _, found := strings.Cut(version, ".")
	majorNum, convErr := strconv.Atoi(major)
	if !found || convErr != nil {
		return fmt.Errorf("could not parse ncu version %q", version)
	}
	if majorNum < MinNcuMajor {
		return fmt.Errorf(
			"ncu %s is too old, version %d or newer is required\nupgrade with: npm install -g npm-check-updates",
			version, MinNcuMajor)
	}
	return nil
}

// ScanGlobal checks globally installed packages. Installed versions come from
// `npm ls -g`; if that lookup fails, packages degrade to an unknown current
// version (severity Other) instead of failing the source.
func (s Scanner) ScanGlobal(ctx context.Context) ([]Package, error) {
	out, err := s.runner.Run(ctx, "", "ncu", "-g", "--jsonUpgraded")
	if err != nil {
		return nil, fmt.Errorf("ncu -g failed: %w", err)
	}
	upgraded, err := parseUpgradedMap(out)
	if err != nil {
		return nil, fmt.Errorf("parsing ncu -g output: %w", err)
	}

	installed := s.globalInstalledVersions(ctx)
	return buildPackages(upgraded, installed), nil
}

// ScanPath scans one registered path, choosing plain or deep mode via detect.
func (s Scanner) ScanPath(ctx context.Context, dir string) ([]Project, error) {
	if detect.ScanMode(dir) == detect.ModeDeep {
		return s.scanDeep(ctx, dir)
	}
	return s.scanSingle(ctx, dir)
}

func (s Scanner) scanSingle(ctx context.Context, dir string) ([]Project, error) {
	pkgFile := filepath.Join(dir, "package.json")
	out, err := s.runner.Run(ctx, dir, "ncu", "--jsonUpgraded", "--packageFile", pkgFile)
	if err != nil {
		return nil, fmt.Errorf("ncu failed for %s: %w", dir, err)
	}
	upgraded, err := parseUpgradedMap(out)
	if err != nil {
		return nil, fmt.Errorf("parsing ncu output for %s: %w", dir, err)
	}
	return []Project{buildProject(dir, ".", upgraded)}, nil
}

func (s Scanner) scanDeep(ctx context.Context, root string) ([]Project, error) {
	out, err := s.runner.Run(ctx, root, "ncu", "--deep", "--jsonUpgraded")
	if err != nil {
		return nil, fmt.Errorf("ncu --deep failed for %s: %w", root, err)
	}

	var perFile map[string]map[string]string
	if err := json.Unmarshal(out, &perFile); err != nil {
		return nil, fmt.Errorf("parsing ncu --deep output for %s: %w", root, err)
	}

	projects := make([]Project, 0, len(perFile))
	for pkgFile, upgraded := range perFile {
		dir := filepath.Dir(resolveUnder(root, pkgFile))
		projects = append(projects, buildProject(dir, labelFor(root, dir), upgraded))
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Label < projects[j].Label })
	return projects, nil
}

// globalInstalledVersions returns installed global versions, tolerating a
// non-zero npm exit as long as stdout holds valid JSON (npm ls does that when
// it finds extraneous packages). Any hard failure yields an empty map.
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

func parseUpgradedMap(out []byte) (map[string]string, error) {
	var upgraded map[string]string
	if err := json.Unmarshal(out, &upgraded); err != nil {
		return nil, err
	}
	return upgraded, nil
}

func buildProject(dir, label string, upgraded map[string]string) Project {
	current := manifestVersions(filepath.Join(dir, "package.json"))
	packages := buildPackages(upgraded, current)
	return Project{
		Dir:      dir,
		Label:    label,
		PM:       detect.PackageManagerFor(dir),
		Packages: packages,
		Counters: countSeverities(packages),
	}
}

func buildPackages(upgraded, current map[string]string) []Package {
	packages := make([]Package, 0, len(upgraded))
	for name, next := range upgraded {
		cur := current[name]
		packages = append(packages, Package{
			Name:     name,
			Current:  cur,
			New:      next,
			Severity: semver.Classify(cur, next),
		})
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Name < packages[j].Name })
	return packages
}

func countSeverities(packages []Package) semver.Counters {
	severities := make([]semver.Severity, len(packages))
	for i, p := range packages {
		severities[i] = p.Severity
	}
	return semver.Count(severities)
}

// manifestVersions reads dependency versions declared in a package.json.
// A missing or malformed manifest yields an empty map (severity degrades to
// Other), never an error.
func manifestVersions(pkgFile string) map[string]string {
	data, err := os.ReadFile(pkgFile)
	if err != nil {
		return nil
	}
	var manifest struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return nil
	}
	versions := make(map[string]string,
		len(manifest.Dependencies)+len(manifest.DevDependencies)+len(manifest.OptionalDependencies))
	for _, group := range []map[string]string{
		manifest.Dependencies, manifest.DevDependencies, manifest.OptionalDependencies,
	} {
		maps.Copy(versions, group)
	}
	return versions
}

// resolveUnder anchors a possibly-relative package-file path to root.
func resolveUnder(root, pkgFile string) string {
	if filepath.IsAbs(pkgFile) {
		return pkgFile
	}
	return filepath.Join(root, pkgFile)
}

func labelFor(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return dir
	}
	return rel
}
