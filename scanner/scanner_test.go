package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/registry"
	"github.com/luchrv/lazyncu/semver"
)

// fakeRunner replies with canned responses keyed by "name arg1 arg2 ...".
type fakeRunner struct {
	responses map[string]fakeResponse
	calls     []string
}

type fakeResponse struct {
	stdout []byte
	err    error
}

func (f *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	if resp, ok := f.responses[key]; ok {
		return resp.stdout, resp.err
	}
	return nil, errors.New("fakeRunner: unexpected command " + key)
}

// fakeRegistry replies with canned metadata or an error per package. Publish
// times come from the metadata itself unless times overrides them.
type fakeRegistry struct {
	meta  map[string]registry.Metadata
	errs  map[string]error
	times map[string]map[string]string
	// minReleaseAge is the raw npm setting answered for every dir.
	minReleaseAge string
}

func (f *fakeRegistry) MinReleaseAge(context.Context, string) (string, error) {
	return f.minReleaseAge, nil
}

func (f *fakeRegistry) PublishTimes(_ context.Context, _, pkg string) (map[string]string, error) {
	if t, ok := f.times[pkg]; ok {
		return t, nil
	}
	return map[string]string{}, nil
}

func (f *fakeRegistry) Fetch(_ context.Context, _, pkg string) (registry.Metadata, error) {
	if err, ok := f.errs[pkg]; ok {
		return registry.Metadata{}, err
	}
	if m, ok := f.meta[pkg]; ok {
		return m, nil
	}
	return registry.Metadata{}, errors.New("fakeRegistry: no metadata for " + pkg)
}

func md(latest string, versions ...registry.Version) registry.Metadata {
	return registry.Metadata{DistTags: map[string]string{"latest": latest}, Versions: versions}
}

func v(version string) registry.Version { return registry.Version{Version: version} }

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func byName(pkgs []Package) map[string]Package {
	out := make(map[string]Package, len(pkgs))
	for _, p := range pkgs {
		out[p.Name] = p
	}
	return out
}

// --- Global scan ---

func TestScanGlobalUsesNdAndRegistry(t *testing.T) {
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm ls -g --depth=0 --json": {stdout: []byte(
			`{"dependencies":{"typescript":{"version":"5.5.0"},"npm-check-updates":{"version":"18.0.1"}}}`)},
	}}
	reg := &fakeRegistry{meta: map[string]registry.Metadata{
		"typescript":        md("5.6.2", v("5.5.0"), v("5.6.2")),
		"npm-check-updates": md("18.1.0", v("18.0.1"), v("18.1.0")),
	}}

	pkgs, err := New(r, reg).ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal() error = %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("returned %d packages, want 2", len(pkgs))
	}
	ts := byName(pkgs)["typescript"]
	if ts.Current != "5.5.0" || ts.New != "5.6.2" || ts.Severity != semver.Minor {
		t.Errorf("typescript = %+v, want current 5.5.0, new 5.6.2, minor", ts)
	}
}

func TestScanGlobalNpmLsFailureDegrades(t *testing.T) {
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm ls -g --depth=0 --json": {err: errors.New("npm ls exploded")},
	}}
	pkgs, err := New(r, &fakeRegistry{}).ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal() error = %v, want degraded success", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("pkgs = %+v, want none when npm ls fails", pkgs)
	}
}

func TestScanGlobalNpmLsExitCodeWithValidJSONIsUsed(t *testing.T) {
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm ls -g --depth=0 --json": {
			stdout: []byte(`{"dependencies":{"typescript":{"version":"5.5.0"}}}`),
			err:    errors.New("exit status 1"),
		},
	}}
	reg := &fakeRegistry{meta: map[string]registry.Metadata{"typescript": md("5.6.2", v("5.5.0"), v("5.6.2"))}}

	pkgs, err := New(r, reg).ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal() error = %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Current != "5.5.0" {
		t.Errorf("pkgs = %+v, want typescript parsed despite non-zero exit", pkgs)
	}
}

func TestScanGlobalRegistryFailureIsIsolated(t *testing.T) {
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm ls -g --depth=0 --json": {stdout: []byte(
			`{"dependencies":{"good":{"version":"1.0.0"},"bad":{"version":"1.0.0"}}}`)},
	}}
	reg := &fakeRegistry{
		meta: map[string]registry.Metadata{"good": md("1.1.0", v("1.0.0"), v("1.1.0"))},
		errs: map[string]error{"bad": errors.New("boom")},
	}

	pkgs, err := New(r, reg).ScanGlobal(context.Background())
	if err != nil {
		t.Fatalf("ScanGlobal() error = %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "good" {
		t.Errorf("pkgs = %+v, want only the healthy dependency", pkgs)
	}
}

// --- Single project scan ---

func TestScanPathSingleProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"),
		`{"name":"app","dependencies":{"express":"^4.18.0"},"devDependencies":{"vitest":"^1.0.0"},"engines":{"node":">=18"}}`)
	writeFile(t, filepath.Join(dir, ".nvmrc"), "18.19.0\n")
	writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "")
	reg := &fakeRegistry{meta: map[string]registry.Metadata{
		"express": md("5.1.0", v("4.18.0"), v("5.1.0")),
		"vitest":  md("1.2.0", v("1.0.0"), v("1.2.0")),
	}}

	projects, err := New(&fakeRunner{}, reg).ScanPath(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1", len(projects))
	}
	p := projects[0]
	if p.Label != "." || p.Dir != dir || p.PM != detect.Pnpm {
		t.Errorf("project = {Label:%q Dir:%q PM:%q}, want {. %s pnpm}", p.Label, p.Dir, p.PM, dir)
	}
	if e := byName(p.Packages)["express"]; e.Current != "^4.18.0" || e.New != "5.1.0" || e.Severity != semver.Major {
		t.Errorf("express = %+v, want current ^4.18.0, new 5.1.0, major", e)
	}
	if p.Counters != (semver.Counters{Major: 1, Minor: 1}) {
		t.Errorf("Counters = %+v, want {Major:1 Minor:1}", p.Counters)
	}
	if p.Nvmrc != "18.19.0" || p.EnginesNode != ">=18" {
		t.Errorf("node context = {Nvmrc:%q EnginesNode:%q}, want {18.19.0 >=18}", p.Nvmrc, p.EnginesNode)
	}
}

func TestScanPathSingleUpToDate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app","dependencies":{}}`)

	projects, err := New(&fakeRunner{}, &fakeRegistry{}).ScanPath(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	if len(projects) != 1 || len(projects[0].Packages) != 0 || projects[0].Counters.Total() != 0 {
		t.Errorf("up-to-date project should have zero packages/counters, got %+v", projects[0])
	}
}

func TestScanPathSingleEnginesFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"),
		`{"name":"app","dependencies":{"lodash":"^1.0.0"},"engines":{"node":">=16 <17"}}`)
	reg := &fakeRegistry{meta: map[string]registry.Metadata{
		"lodash": md("3.0.0", v("1.0.0"), v("2.0.0"), registry.Version{Version: "3.0.0", EnginesNode: ">=20"}),
	}}
	// 2.0.0 has no engines (allowed); 3.0.0 needs node >=20 (excluded)
	reg.meta["lodash"] = registry.Metadata{
		DistTags: map[string]string{"latest": "3.0.0"},
		Versions: []registry.Version{
			{Version: "1.0.0"},
			{Version: "2.0.0", EnginesNode: ">=16"},
			{Version: "3.0.0", EnginesNode: ">=20"},
		},
	}

	projects, err := New(&fakeRunner{}, reg).ScanPath(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	pkgs := byName(projects[0].Packages)
	if pkgs["lodash"].New != "2.0.0" {
		t.Errorf("lodash new = %q, want 2.0.0 (newest satisfying engines)", pkgs["lodash"].New)
	}
}

func TestScanPathSingleRegistryFailureIsolated(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"),
		`{"name":"app","dependencies":{"a":"^1.0.0","b":"^1.0.0"}}`)
	reg := &fakeRegistry{
		meta: map[string]registry.Metadata{"b": md("1.1.0", v("1.0.0"), v("1.1.0"))},
		errs: map[string]error{"a": errors.New("registry down")},
	}

	projects, err := New(&fakeRunner{}, reg).ScanPath(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanPath() error = %v, want success despite one failure", err)
	}
	if len(projects[0].Packages) != 1 || projects[0].Packages[0].Name != "b" {
		t.Errorf("packages = %+v, want only b", projects[0].Packages)
	}
}

func TestScanPathSingleRejectViaNcurc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"),
		`{"name":"app","dependencies":{"@types/node":"^1.0.0","express":"^4.0.0"}}`)
	writeFile(t, filepath.Join(dir, ".ncurc.json"), `{"reject":["@types/*"]}`)
	reg := &fakeRegistry{meta: map[string]registry.Metadata{
		"@types/node": md("1.1.0", v("1.0.0"), v("1.1.0")),
		"express":     md("5.0.0", v("4.0.0"), v("5.0.0")),
	}}

	projects, err := New(&fakeRunner{}, reg).ScanPath(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	if pkgs := byName(projects[0].Packages); pkgs["@types/node"].Name != "" || pkgs["express"].Name == "" {
		t.Errorf("packages = %+v, want @types/node rejected and express present", projects[0].Packages)
	}
}

func TestScanPathSurfacesNcurcWarnings(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app"}`)
	writeFile(t, filepath.Join(dir, ".ncurc"), "peer: true\n")

	projects, err := New(&fakeRunner{}, &fakeRegistry{}).ScanPath(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	if len(projects[0].Warnings) == 0 {
		t.Error("Project.Warnings empty, want the out-of-scope .ncurc warning")
	}
}

func TestScanPathFolderIsRefused(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "api", "package.json"), `{"dependencies":{"express":"^4.18.0"}}`)

	projects, err := New(&fakeRunner{}, &fakeRegistry{}).ScanPath(context.Background(), root)
	if !errors.Is(err, ErrFolderNotScannable) {
		t.Fatalf("ScanPath() error = %v, want ErrFolderNotScannable", err)
	}
	if projects != nil {
		t.Errorf("projects = %v, want nil", projects)
	}
}

// --- Deep scan ---

func TestScanPathDeepDiscoversEveryManifest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"name":"mono","workspaces":["packages/*"]}`)
	writeFile(t, filepath.Join(root, "packages", "api", "package.json"), `{"dependencies":{"express":"^4.18.0"}}`)
	writeFile(t, filepath.Join(root, "packages", "api", "package-lock.json"), "{}")
	writeFile(t, filepath.Join(root, "packages", "web", "package.json"), `{"dependencies":{"react":"^18.2.0"}}`)
	writeFile(t, filepath.Join(root, "packages", "web", "yarn.lock"), "")
	reg := &fakeRegistry{meta: map[string]registry.Metadata{
		"express": md("5.1.0", v("4.18.0"), v("5.1.0")),
		"react":   md("18.3.1", v("18.2.0"), v("18.3.1")),
	}}

	projects, err := New(&fakeRunner{}, reg).ScanPath(context.Background(), root)
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	byLabel := map[string]Project{}
	for _, p := range projects {
		byLabel[p.Label] = p
	}
	api, ok := byLabel[filepath.Join("packages", "api")]
	if !ok {
		t.Fatalf("missing project labeled packages/api; labels: %v", labels(projects))
	}
	if api.PM != detect.Npm || api.Dir != filepath.Join(root, "packages", "api") {
		t.Errorf("api = {PM:%q Dir:%q}, want npm, %s", api.PM, api.Dir, filepath.Join(root, "packages", "api"))
	}
	if e := byName(api.Packages)["express"]; e.Current != "^4.18.0" || e.Severity != semver.Major {
		t.Errorf("api express = %+v, want current ^4.18.0 major", e)
	}
	if web := byLabel[filepath.Join("packages", "web")]; web.PM != detect.Yarn {
		t.Errorf("web.PM = %q, want yarn", web.PM)
	}
	if _, ok := byLabel["."]; !ok {
		t.Errorf("root manifest not discovered; labels: %v", labels(projects))
	}
}

func TestScanPathSingleCooldownExcludesFreshVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app","dependencies":{"lodash":"^1.0.0"}}`)
	now := time.Now()
	reg := &fakeRegistry{meta: map[string]registry.Metadata{
		"lodash": {
			DistTags: map[string]string{"latest": "2.0.0"},
			Versions: []registry.Version{
				{Version: "1.0.0"},
				{Version: "2.0.0", Time: now.Add(-24 * time.Hour).Format(time.RFC3339)},
				{Version: "1.5.0", Time: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)},
			},
		},
	}}
	reg.minReleaseAge = "7"

	projects, err := New(&fakeRunner{}, reg).ScanPath(context.Background(), dir)
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	if got := byName(projects[0].Packages)["lodash"].New; got != "1.5.0" {
		t.Errorf("lodash new = %q, want 1.5.0 (2.0.0 within cooldown)", got)
	}
}

func TestManifestVersionsIncludesPackageManager(t *testing.T) {
	dir := t.TempDir()
	pkgFile := filepath.Join(dir, "package.json")
	writeFile(t, pkgFile, `{"name":"app","packageManager":"pnpm@9.0.0","dependencies":{"x":"1.0.0"}}`)

	got := manifestVersions(pkgFile)
	if got["pnpm"] != "9.0.0" {
		t.Errorf("manifestVersions()[pnpm] = %q, want 9.0.0", got["pnpm"])
	}
	if got["x"] != "1.0.0" {
		t.Errorf("manifestVersions()[x] = %q, want 1.0.0", got["x"])
	}
}

func TestParseCooldown(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0}, {"undefined", 0}, {"7", 7}, {"7d", 7}, {"48h", 2}, {"60m", 0}, {"bogus", 0}, {"-3", 0},
	}
	for _, tt := range tests {
		if got := parseCooldown(tt.in); got != tt.want {
			t.Errorf("parseCooldown(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func labels(projects []Project) []string {
	out := make([]string, len(projects))
	for i, p := range projects {
		out[i] = p.Label
	}
	return out
}
