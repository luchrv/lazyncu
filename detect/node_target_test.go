package detect

import (
	"os"
	"path/filepath"
	"testing"
)

// writeNested creates a file (and its parents) under root.
func writeNested(t *testing.T, root string, parts ...string) {
	t.Helper()
	path := filepath.Join(append([]string{root}, parts...)...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHasNodeProject(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root string)
		want  bool
	}{
		{
			name:  "package.json at root",
			setup: func(t *testing.T, root string) { writeNested(t, root, "package.json") },
			want:  true,
		},
		{
			name:  "nested project one level down",
			setup: func(t *testing.T, root string) { writeNested(t, root, "api", "package.json") },
			want:  true,
		},
		{
			name:  "monorepo packages layout",
			setup: func(t *testing.T, root string) { writeNested(t, root, "packages", "core", "package.json") },
			want:  true,
		},
		{
			name:  "empty directory",
			setup: func(t *testing.T, root string) {},
			want:  false,
		},
		{
			name:  "no package.json anywhere",
			setup: func(t *testing.T, root string) { writeNested(t, root, "src", "main.go") },
			want:  false,
		},
		{
			name: "package.json beyond search depth",
			setup: func(t *testing.T, root string) {
				writeNested(t, root, "a", "b", "c", "package.json")
			},
			want: false,
		},
		{
			name: "package.json only inside node_modules",
			setup: func(t *testing.T, root string) {
				writeNested(t, root, "node_modules", "left-pad", "package.json")
			},
			want: false,
		},
		{
			name: "package.json only inside hidden directory",
			setup: func(t *testing.T, root string) {
				writeNested(t, root, ".cache", "package.json")
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			tt.setup(t, root)

			got := HasNodeProject(root)

			if got != tt.want {
				t.Errorf("HasNodeProject(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestHasNodeProjectMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	if HasNodeProject(missing) {
		t.Errorf("HasNodeProject(missing dir) = true, want false")
	}
}
