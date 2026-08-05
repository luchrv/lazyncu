package launch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luchrv/lazyncu/config"
)

// nodeDir creates a temp directory containing a package.json.
func nodeDir(t *testing.T, parent string, parts ...string) string {
	t.Helper()
	dir := filepath.Join(append([]string{parent}, parts...)...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkg, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestResolve(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "dot resolves to cwd", raw: ".", want: cwd},
		{name: "dotdot resolves to parent", raw: "..", want: filepath.Dir(cwd)},
		{name: "relative resolves against cwd", raw: "sub/dir", want: filepath.Join(cwd, "sub", "dir")},
		{name: "tilde expands to home", raw: "~/projects", want: filepath.Join(home, "projects")},
		{name: "bare tilde is home", raw: "~", want: home},
		{name: "absolute stays and cleans", raw: "/a/b/../c", want: filepath.Clean("/a/c")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.raw)

			if err != nil {
				t.Fatalf("Resolve(%q) error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{name: "direct child", parent: "/projects", child: "/projects/api", want: true},
		{name: "deep descendant", parent: "/projects", child: "/projects/a/b/c", want: true},
		{name: "equal is not containment", parent: "/projects", child: "/projects", want: false},
		{name: "sibling name prefix trap", parent: "/projects", child: "/projects-old/api", want: false},
		{name: "reversed relation", parent: "/projects/api", child: "/projects", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.parent, tt.child)

			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}

func TestComparableResolvesSymlinks(t *testing.T) {
	base := t.TempDir()
	real := nodeDir(t, base, "real")
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if Comparable(link) != Comparable(real) {
		t.Errorf("Comparable(%q) = %q, want same as Comparable(%q) = %q",
			link, Comparable(link), real, Comparable(real))
	}
}

func TestClassify(t *testing.T) {
	registered := []config.Path{{Path: "/projects"}, {Path: "/other/web"}}

	tests := []struct {
		name       string
		registered []config.Path
		target     string
		wantKind   Kind
		wantSource string
	}{
		{
			name:       "equal to registered path",
			registered: registered,
			target:     "/projects",
			wantKind:   KindEqual,
			wantSource: "/projects",
		},
		{
			name:       "contained in registered path",
			registered: registered,
			target:     "/projects/api",
			wantKind:   KindContained,
			wantSource: "/projects",
		},
		{
			name:       "parent of registered path",
			registered: registered,
			target:     "/other",
			wantKind:   KindParent,
			wantSource: "/other",
		},
		{
			name:       "unrelated new path",
			registered: registered,
			target:     "/elsewhere/app",
			wantKind:   KindNew,
			wantSource: "/elsewhere/app",
		},
		{
			name:       "sibling prefix is new, not contained",
			registered: registered,
			target:     "/projects-old/api",
			wantKind:   KindNew,
			wantSource: "/projects-old/api",
		},
		{
			name:       "no registered paths",
			registered: nil,
			target:     "/anything",
			wantKind:   KindNew,
			wantSource: "/anything",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.registered, tt.target)

			if got.Kind != tt.wantKind {
				t.Errorf("Classify(%q).Kind = %v, want %v", tt.target, got.Kind, tt.wantKind)
			}
			if got.Source != tt.wantSource {
				t.Errorf("Classify(%q).Source = %q, want %q", tt.target, got.Source, tt.wantSource)
			}
			if got.TargetDir != tt.target {
				t.Errorf("Classify(%q).TargetDir = %q, want %q", tt.target, got.TargetDir, tt.target)
			}
		})
	}
}

func TestClassifyParentListsAllCoveredChildren(t *testing.T) {
	registered := []config.Path{
		{Path: "/projects/api"},
		{Path: "/projects/web"},
		{Path: "/other"},
	}

	got := Classify(registered, "/projects")

	if got.Kind != KindParent {
		t.Fatalf("Kind = %v, want KindParent", got.Kind)
	}
	expectedChildren := []string{"/projects/api", "/projects/web"}
	if len(got.CoveredChildren) != len(expectedChildren) {
		t.Fatalf("CoveredChildren = %v, want %v", got.CoveredChildren, expectedChildren)
	}
	for i, want := range expectedChildren {
		if got.CoveredChildren[i] != want {
			t.Errorf("CoveredChildren[%d] = %q, want %q", i, got.CoveredChildren[i], want)
		}
	}
}

func TestPrepareNewPathPersists(t *testing.T) {
	base := t.TempDir()
	target := nodeDir(t, base, "app")
	cfgPath := filepath.Join(base, "config.toml")
	cfg := config.Config{TimeoutMS: 1000}

	updated, intent, err := Prepare(cfg, cfgPath, target)

	if err != nil {
		t.Fatalf("Prepare error: %v", err)
	}
	if intent.Kind != KindNew {
		t.Errorf("Kind = %v, want KindNew", intent.Kind)
	}
	if len(updated.Paths) != 1 || updated.Paths[0].Path != target {
		t.Errorf("updated.Paths = %+v, want [%s]", updated.Paths, target)
	}
	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("reloading config: %v", err)
	}
	if len(loaded.Paths) != 1 || loaded.Paths[0].Path != target {
		t.Errorf("persisted Paths = %+v, want [%s]", loaded.Paths, target)
	}
}

func TestPrepareParentPersistsAndListsChildren(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "projects")
	child := nodeDir(t, parent, "api")
	cfgPath := filepath.Join(base, "config.toml")
	cfg := config.Config{Paths: []config.Path{{Path: child}}}

	updated, intent, err := Prepare(cfg, cfgPath, parent)

	if err != nil {
		t.Fatalf("Prepare error: %v", err)
	}
	if intent.Kind != KindParent {
		t.Errorf("Kind = %v, want KindParent", intent.Kind)
	}
	if len(intent.CoveredChildren) != 1 || intent.CoveredChildren[0] != child {
		t.Errorf("CoveredChildren = %v, want [%s]", intent.CoveredChildren, child)
	}
	if len(updated.Paths) != 2 {
		t.Errorf("updated.Paths = %+v, want child plus parent", updated.Paths)
	}
}

func TestPrepareContainedLeavesConfigUntouched(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "projects")
	child := nodeDir(t, parent, "api")
	cfgPath := filepath.Join(base, "config.toml")
	cfg := config.Config{Paths: []config.Path{{Path: parent}}}

	updated, intent, err := Prepare(cfg, cfgPath, child)

	if err != nil {
		t.Fatalf("Prepare error: %v", err)
	}
	if intent.Kind != KindContained {
		t.Errorf("Kind = %v, want KindContained", intent.Kind)
	}
	if intent.Source != parent {
		t.Errorf("Source = %q, want %q", intent.Source, parent)
	}
	if len(updated.Paths) != 1 {
		t.Errorf("updated.Paths = %+v, want unchanged single entry", updated.Paths)
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Errorf("config file written for contained target, want untouched")
	}
}

func TestPrepareRejectsInvalidTargets(t *testing.T) {
	base := t.TempDir()
	emptyDir := filepath.Join(base, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(base, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		target  string
		wantErr string
	}{
		{name: "missing path", target: filepath.Join(base, "nope"), wantErr: "not accessible"},
		{name: "not a directory", target: file, wantErr: "not a directory"},
		{name: "no node project", target: emptyDir, wantErr: "no Node project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Prepare(config.Config{}, filepath.Join(base, "config.toml"), tt.target)

			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Prepare(%q) error = %v, want containing %q", tt.target, err, tt.wantErr)
			}
		})
	}
}
