package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/semver"
)

// fakeRunner replies with canned responses keyed by "name arg1 arg2 ...".
type fakeRunner struct {
	responses map[string]fakeResponse
	calls     []string
}

type fakeResponse struct {
	stdout   []byte
	err      error
	blockCtx bool // simulate a hang: return only when ctx is done
}

func (f *fakeRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	resp, ok := f.responses[key]
	if !ok {
		return nil, errors.New("fakeRunner: unexpected command " + key)
	}
	if resp.blockCtx {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return resp.stdout, resp.err
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- Preflight ---

func TestPreflightMissingBinary(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --version": {err: errors.New(`exec: "ncu": executable file not found in $PATH`)},
	}}

	// Act
	err := New(r).Preflight(context.Background())

	// Assert
	if err == nil {
		t.Fatal("Preflight() = nil, want error for missing ncu")
	}
	if !strings.Contains(err.Error(), "npm install -g npm-check-updates") {
		t.Errorf("Preflight() error %q must include the install command", err)
	}
}

func TestPreflightVersionTooOld(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --version": {stdout: []byte("17.1.3\n")},
	}}

	// Act
	err := New(r).Preflight(context.Background())

	// Assert
	if err == nil {
		t.Fatal("Preflight() = nil, want error for ncu < 18")
	}
	if !strings.Contains(err.Error(), "18") {
		t.Errorf("Preflight() error %q must mention the minimum version", err)
	}
}

func TestPreflightOK(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --version": {stdout: []byte("18.0.1\n")},
	}}

	// Act & Assert
	if err := New(r).Preflight(context.Background()); err != nil {
		t.Fatalf("Preflight() = %v, want nil", err)
	}
}

// --- Global scan ---

func TestScanGlobalMergesInstalledVersions(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu -g --jsonUpgraded": {stdout: []byte(`{"typescript":"5.6.2","npm-check-updates":"18.1.0"}`)},
		"npm ls -g --depth=0 --json": {stdout: []byte(
			`{"dependencies":{"typescript":{"version":"5.5.0"},"npm-check-updates":{"version":"18.0.1"}}}`)},
	}}

	// Act
	pkgs, err := New(r).ScanGlobal(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("ScanGlobal() error = %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("ScanGlobal() returned %d packages, want 2", len(pkgs))
	}
	byName := map[string]Package{}
	for _, p := range pkgs {
		byName[p.Name] = p
	}
	ts := byName["typescript"]
	if ts.Current != "5.5.0" || ts.New != "5.6.2" || ts.Severity != semver.Minor {
		t.Errorf("typescript = %+v, want current 5.5.0, new 5.6.2, minor", ts)
	}
}

func TestScanGlobalNpmLsFailureDegrades(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu -g --jsonUpgraded":      {stdout: []byte(`{"typescript":"5.6.2"}`)},
		"npm ls -g --depth=0 --json": {err: errors.New("npm ls exploded")},
	}}

	// Act
	pkgs, err := New(r).ScanGlobal(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("ScanGlobal() error = %v, want degraded success", err)
	}
	if len(pkgs) != 1 || pkgs[0].Current != "" || pkgs[0].Severity != semver.Other {
		t.Errorf("pkgs = %+v, want typescript with unknown current and other severity", pkgs)
	}
}

func TestScanGlobalNpmLsExitCodeWithValidJSONIsUsed(t *testing.T) {
	// Arrange: npm ls -g exits non-zero (e.g. extraneous packages) but emits JSON
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu -g --jsonUpgraded": {stdout: []byte(`{"typescript":"5.6.2"}`)},
		"npm ls -g --depth=0 --json": {
			stdout: []byte(`{"dependencies":{"typescript":{"version":"5.5.0"}}}`),
			err:    errors.New("exit status 1"),
		},
	}}

	// Act
	pkgs, err := New(r).ScanGlobal(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("ScanGlobal() error = %v", err)
	}
	if pkgs[0].Current != "5.5.0" {
		t.Errorf("Current = %q, want 5.5.0 parsed despite non-zero exit", pkgs[0].Current)
	}
}

func TestScanGlobalNcuFails(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu -g --jsonUpgraded": {err: errors.New("exit status 2")},
	}}

	// Act
	_, err := New(r).ScanGlobal(context.Background())

	// Assert
	if err == nil {
		t.Fatal("ScanGlobal() = nil, want error when ncu fails")
	}
}

// --- Single project scan ---

func TestScanPathSingleProject(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"),
		`{"name":"app","dependencies":{"express":"^4.18.0"},"devDependencies":{"vitest":"^1.0.0"}}`)
	writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "")
	pkgFile := filepath.Join(dir, "package.json")
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --jsonUpgraded --packageFile " + pkgFile: {stdout: []byte(`{"express":"^5.1.0","vitest":"^1.2.0"}`)},
	}}

	// Act
	projects, err := New(r).ScanPath(context.Background(), dir)

	// Assert
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
	if len(p.Packages) != 2 {
		t.Fatalf("got %d packages, want 2", len(p.Packages))
	}
	byName := map[string]Package{}
	for _, pkg := range p.Packages {
		byName[pkg.Name] = pkg
	}
	if e := byName["express"]; e.Current != "^4.18.0" || e.New != "^5.1.0" || e.Severity != semver.Major {
		t.Errorf("express = %+v, want current ^4.18.0, new ^5.1.0, major", e)
	}
	if p.Counters != (semver.Counters{Major: 1, Minor: 1}) {
		t.Errorf("Counters = %+v, want {Major:1 Minor:1}", p.Counters)
	}
}

func TestScanPathSingleUpToDate(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app","dependencies":{}}`)
	pkgFile := filepath.Join(dir, "package.json")
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --jsonUpgraded --packageFile " + pkgFile: {stdout: []byte(`{}`)},
	}}

	// Act
	projects, err := New(r).ScanPath(context.Background(), dir)

	// Assert
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	if len(projects) != 1 || len(projects[0].Packages) != 0 || projects[0].Counters.Total() != 0 {
		t.Errorf("up-to-date project should have zero packages/counters, got %+v", projects[0])
	}
}

// --- Deep scan ---

func TestScanPathDeepFolderOfProjects(t *testing.T) {
	// Arrange: a folder (no root package.json) with two projects
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "api", "package.json"), `{"dependencies":{"express":"^4.18.0"}}`)
	writeFile(t, filepath.Join(root, "api", "package-lock.json"), "{}")
	writeFile(t, filepath.Join(root, "web", "package.json"), `{"dependencies":{"react":"^18.2.0"}}`)
	writeFile(t, filepath.Join(root, "web", "yarn.lock"), "")
	deepJSON := `{
		"api/package.json": {"express":"^5.1.0"},
		"web/package.json": {"react":"^18.3.1"}
	}`
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --deep --jsonUpgraded": {stdout: []byte(deepJSON)},
	}}

	// Act
	projects, err := New(r).ScanPath(context.Background(), root)

	// Assert
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(projects))
	}
	byLabel := map[string]Project{}
	for _, p := range projects {
		byLabel[p.Label] = p
	}
	api, ok := byLabel["api"]
	if !ok {
		t.Fatalf("missing project labeled 'api'; labels: %v", labels(projects))
	}
	if api.PM != detect.Npm || api.Dir != filepath.Join(root, "api") {
		t.Errorf("api = {PM:%q Dir:%q}, want npm, %s", api.PM, api.Dir, filepath.Join(root, "api"))
	}
	if api.Packages[0].Current != "^4.18.0" || api.Packages[0].Severity != semver.Major {
		t.Errorf("api express = %+v, want current ^4.18.0 major", api.Packages[0])
	}
	if web := byLabel["web"]; web.PM != detect.Yarn {
		t.Errorf("web.PM = %q, want yarn", web.PM)
	}
}

func TestScanPathDeepMonorepoWithAbsoluteKeys(t *testing.T) {
	// Arrange: monorepo root; ncu may emit absolute package-file paths
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"name":"root","workspaces":["packages/*"]}`)
	writeFile(t, filepath.Join(root, "pnpm-lock.yaml"), "")
	writeFile(t, filepath.Join(root, "packages", "core", "package.json"), `{"dependencies":{"lodash":"^4.17.20"}}`)
	deepJSON := `{"` + filepath.Join(root, "packages", "core", "package.json") + `": {"lodash":"^4.17.21"}}`
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --deep --jsonUpgraded": {stdout: []byte(deepJSON)},
	}}

	// Act
	projects, err := New(r).ScanPath(context.Background(), root)

	// Assert
	if err != nil {
		t.Fatalf("ScanPath() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("got %d projects, want 1", len(projects))
	}
	p := projects[0]
	if p.Label != filepath.Join("packages", "core") {
		t.Errorf("Label = %q, want packages/core", p.Label)
	}
	if p.Packages[0].Severity != semver.Patch {
		t.Errorf("lodash severity = %v, want patch", p.Packages[0].Severity)
	}
}

// --- Failure isolation ---

func TestScanPathNcuNonZeroExit(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app"}`)
	pkgFile := filepath.Join(dir, "package.json")
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --jsonUpgraded --packageFile " + pkgFile: {err: errors.New("exit status 2")},
	}}

	// Act
	_, err := New(r).ScanPath(context.Background(), dir)

	// Assert
	if err == nil {
		t.Fatal("ScanPath() = nil, want error on ncu failure")
	}
}

func TestScanPathMalformedJSONNoPanic(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app"}`)
	pkgFile := filepath.Join(dir, "package.json")
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --jsonUpgraded --packageFile " + pkgFile: {stdout: []byte(`{{{not json`)},
	}}

	// Act
	_, err := New(r).ScanPath(context.Background(), dir)

	// Assert
	if err == nil {
		t.Fatal("ScanPath() = nil, want parse error, and must not panic")
	}
}

func TestScanPathTimeout(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"app"}`)
	pkgFile := filepath.Join(dir, "package.json")
	r := &fakeRunner{responses: map[string]fakeResponse{
		"ncu --jsonUpgraded --packageFile " + pkgFile: {blockCtx: true},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Act
	_, err := New(r).ScanPath(ctx, dir)

	// Assert
	if err == nil {
		t.Fatal("ScanPath() = nil, want context error on timeout")
	}
}

func labels(projects []Project) []string {
	out := make([]string, len(projects))
	for i, p := range projects {
		out[i] = p.Label
	}
	return out
}
