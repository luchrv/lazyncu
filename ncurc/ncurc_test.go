package ncurc

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadJSON(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".ncurc.json", `{"reject":["@types/*"],"pre":true,"deprecated":false}`)

	got := Load(dir)

	if len(got.Options.Reject) != 1 || got.Options.Reject[0] != "@types/*" {
		t.Errorf("Reject = %v, want [@types/*]", got.Options.Reject)
	}
	if !got.Options.Pre {
		t.Error("Pre = false, want true")
	}
	if got.Options.AllowDeprecated() {
		t.Error("AllowDeprecated() = true, want false from deprecated:false")
	}
	if len(got.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", got.Warnings)
	}
}

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".ncurc.yml", "reject:\n  - '@types/*'\ntarget: '@next'\n")

	got := Load(dir)

	if got.Options.Reject[0] != "@types/*" {
		t.Errorf("Reject = %v, want [@types/*]", got.Options.Reject)
	}
	tag, ok := got.Options.DistTag()
	if !ok || tag != "next" {
		t.Errorf("DistTag() = %q, %v, want next, true", tag, ok)
	}
}

func TestLoadNoFile(t *testing.T) {
	got := Load(t.TempDir())
	if len(got.Warnings) != 0 || got.Options.Reject != nil {
		t.Errorf("Load(no file) = %+v, want zero result", got)
	}
	if !got.Options.AllowDeprecated() {
		t.Error("AllowDeprecated() default = false, want true")
	}
}

func TestLoadWarnsOnOutOfScopeKey(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".ncurc", "peer: true\nreject:\n  - foo\n")

	got := Load(dir)

	if len(got.Options.Reject) != 1 {
		t.Errorf("supported options not applied: %+v", got.Options)
	}
	if len(got.Warnings) != 1 || got.Warnings[0] == "" {
		t.Fatalf("Warnings = %v, want one warning about peer", got.Warnings)
	}
	if !contains(got.Warnings[0], "peer") {
		t.Errorf("warning %q must name the offending key", got.Warnings[0])
	}
}

func TestLoadWarnsOnExecutableConfig(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, ".ncurc.js", "module.exports = { reject: ['x'] }\n")

	got := Load(dir)

	if len(got.Warnings) != 1 || !contains(got.Warnings[0], ".ncurc.js") {
		t.Errorf("Warnings = %v, want one about .ncurc.js", got.Warnings)
	}
}

func TestLoadNearestSearchesUpward(t *testing.T) {
	root := t.TempDir()
	write(t, root, ".ncurc", "reject:\n  - root-pkg\n")
	child := filepath.Join(root, "packages", "app")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	got := Load(child)

	if len(got.Options.Reject) != 1 || got.Options.Reject[0] != "root-pkg" {
		t.Errorf("Reject = %v, want the root config's [root-pkg]", got.Options.Reject)
	}
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"@types/*", "@types/node", true},
		{"@types/*", "@types/react-dom", true},
		{"@types/*", "express", false},
		{"lodash", "lodash", true},
		{"lodash", "lodash-es", false},
		{"@babel/*", "@babel/core", true},
	}
	for _, tt := range tests {
		if got := MatchesAny([]string{tt.pattern}, tt.name); got != tt.want {
			t.Errorf("MatchesAny(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
	if MatchesAny(nil, "x") {
		t.Error("MatchesAny(nil, x) = true, want false")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
