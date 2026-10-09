package registry

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

// configListJSON mimics `npm config list --json` naming the npmrc files.
func configListJSON(userconfig, globalconfig string, extra string) []byte {
	return []byte(`{
		"registry": "https://registry.npmjs.org/",
		"strict-ssl": true,
		"cafile": null,
		"proxy": null,
		"https-proxy": null,
		"userconfig": "` + userconfig + `",
		"globalconfig": "` + globalconfig + `"` + extra + `
	}`)
}

func writeNpmrc(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveReadsScopesAndCredentials(t *testing.T) {
	// Arrange: user npmrc with a token, project npmrc overriding a scope
	tmp := t.TempDir()
	t.Setenv("ACME_TOKEN", "tok-from-env")
	user := writeNpmrc(t, tmp, "user.npmrc",
		"//npm.acme.test/:_authToken=${ACME_TOKEN}\n//npm.acme.test/private/:_authToken=\"deep-token\"\n")
	project := filepath.Join(tmp, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	writeNpmrc(t, project, ".npmrc", "; project overrides\n//registry.npmjs.org/:username=alice\n//registry.npmjs.org/:_password="+
		base64.StdEncoding.EncodeToString([]byte("s3cret"))+"\n")
	r := &fakeRunner{responses: map[string][]byte{
		"npm config list --json": configListJSON(user, filepath.Join(tmp, "missing-global"),
			`, "@acme:registry": "https://npm.acme.test/private/", "@other:registry": "https://npm.acme.test/"`),
	}}
	resolver := NewResolver(r)
	resolver.env = func() []string { return []string{"npm_config_//env.test/:_authToken=env-token"} }

	// Act
	cfg, err := resolver.Resolve(context.Background(), project)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	acmeURL, acmeAuth := cfg.Endpoint("@acme/pkg")
	otherURL, otherAuth := cfg.Endpoint("@other/pkg")
	defaultURL, defaultAuth := cfg.Endpoint("lodash")

	// Assert
	if acmeURL != "https://npm.acme.test/private/" || acmeAuth.Token != "deep-token" {
		t.Errorf("@acme endpoint = %s %+v, want the deepest nerf-dart token", acmeURL, acmeAuth)
	}
	if otherURL != "https://npm.acme.test/" || otherAuth.Token != "tok-from-env" {
		t.Errorf("@other endpoint = %s %+v, want the host token expanded from the environment", otherURL, otherAuth)
	}
	wantBasic := base64.StdEncoding.EncodeToString([]byte("alice:s3cret"))
	if defaultURL != "https://registry.npmjs.org/" || defaultAuth.Basic != wantBasic {
		t.Errorf("default endpoint = %s %+v, want basic auth from the project npmrc", defaultURL, defaultAuth)
	}
	if cfg.auth["//env.test/"].Token != "env-token" {
		t.Errorf("env credential not read: %+v", cfg.auth)
	}
}

func TestResolveRunsNpmOncePerProcess(t *testing.T) {
	r := &fakeRunner{responses: map[string][]byte{
		"npm config list --json": configListJSON("/nonexistent/user", "/nonexistent/global", `, "min-release-age": "7"`),
	}}
	resolver := NewResolver(r)

	for _, dir := range []string{"/p/a", "/p/a", "/p/b", ""} {
		cfg, err := resolver.Resolve(context.Background(), dir)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.MinReleaseAge != "7" {
			t.Errorf("%q: min-release-age = %q, want 7", dir, cfg.MinReleaseAge)
		}
	}

	if len(r.calls) != 1 {
		t.Errorf("npm config list ran %d times, want 1 (project layers are overlaid locally)", len(r.calls))
	}
}

func TestProjectNpmrcOverridesRegistryAndCooldown(t *testing.T) {
	project := t.TempDir()
	writeNpmrc(t, project, ".npmrc", "registry=https://mirror.test/npm/\nmin-release-age=3d\nstrict-ssl=false\n")
	r := &fakeRunner{responses: map[string][]byte{
		"npm config list --json": configListJSON("/nonexistent/user", "/nonexistent/global", ""),
	}}
	resolver := NewResolver(r)

	cfg, err := resolver.Resolve(context.Background(), project)
	base, _ := resolver.Resolve(context.Background(), "")

	if err != nil {
		t.Fatal(err)
	}
	if url, _ := cfg.Endpoint("x"); url != "https://mirror.test/npm/" || cfg.MinReleaseAge != "3d" || cfg.StrictSSL {
		t.Errorf("project config = %+v, want the .npmrc overrides", cfg)
	}
	if url, _ := base.Endpoint("x"); url != "https://registry.npmjs.org/" || !base.StrictSSL {
		t.Errorf("base config mutated by the project overlay: %+v", base)
	}
}

func TestResolveFailsWithoutRegistry(t *testing.T) {
	r := &fakeRunner{responses: map[string][]byte{"npm config list --json": []byte(`{"strict-ssl": true}`)}}
	if _, err := NewResolver(r).Resolve(context.Background(), ""); err == nil {
		t.Fatal("Resolve() = nil error, want failure when npm reports no registry")
	}
	r = &fakeRunner{responses: map[string][]byte{"npm config list --json": []byte(`npm warn something`)}}
	if _, err := NewResolver(r).Resolve(context.Background(), ""); err == nil {
		t.Fatal("Resolve() = nil error, want failure on non-JSON output")
	}
}

func TestAuthForWalksNerfDarts(t *testing.T) {
	cfg := NpmConfig{Registry: "https://host.test/a/b/", auth: map[string]Auth{
		"//host.test/":   {Token: "root"},
		"//host.test/a/": {Token: "mid"},
	}}
	if _, a := cfg.Endpoint("x"); a.Token != "mid" {
		t.Errorf("token = %q, want the closest prefix (mid)", a.Token)
	}
	cfg.Registry = "https://host.test/"
	if _, a := cfg.Endpoint("x"); a.Token != "root" {
		t.Errorf("token = %q, want root", a.Token)
	}
	cfg.Registry = "https://elsewhere.test/"
	if _, a := cfg.Endpoint("x"); a != (Auth{}) {
		t.Errorf("token = %+v, want none for an unknown host", a)
	}
}
