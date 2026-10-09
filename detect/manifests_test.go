package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestManifestsIncludesRootAndNested(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"name":"root","workspaces":["packages/*"]}`)
	mkdir(t, filepath.Join(root, "packages", "core"))
	writeFile(t, filepath.Join(root, "packages", "core"), "package.json", `{"name":"core"}`)

	got, err := Manifests(root)
	if err != nil {
		t.Fatalf("Manifests() error = %v", err)
	}
	want := []string{".", filepath.Join("packages", "core")}
	if labels := manifestLabels(got); !equal(labels, want) {
		t.Errorf("labels = %v, want %v (root + nested)", labels, want)
	}
}

func TestManifestsSkipsNodeModulesAndStore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"name":"root"}`)
	mkdir(t, filepath.Join(root, "node_modules", "pkg"))
	writeFile(t, filepath.Join(root, "node_modules", "pkg"), "package.json", `{"name":"pkg"}`)
	mkdir(t, filepath.Join(root, ".pnpm-store", "x"))
	writeFile(t, filepath.Join(root, ".pnpm-store", "x"), "package.json", `{"name":"x"}`)

	got, err := Manifests(root)
	if err != nil {
		t.Fatalf("Manifests() error = %v", err)
	}
	if labels := manifestLabels(got); !equal(labels, []string{"."}) {
		t.Errorf("labels = %v, want only [.]", labels)
	}
}

func TestManifestsRootWithoutManifest(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "a"))
	writeFile(t, filepath.Join(root, "a"), "package.json", `{"name":"a"}`)

	got, err := Manifests(root)
	if err != nil {
		t.Fatalf("Manifests() error = %v", err)
	}
	if labels := manifestLabels(got); !equal(labels, []string{"a"}) {
		t.Errorf("labels = %v, want [a]", labels)
	}
}

func manifestLabels(ms []Manifest) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Label
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
