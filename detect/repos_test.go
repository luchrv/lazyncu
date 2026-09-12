package detect

import (
	"os"
	"path/filepath"
	"testing"
)

// mkRepo creates dir (and parents) with a package.json inside.
func mkRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "package.json", `{"name":"`+filepath.Base(dir)+`"}`)
}

func mkDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func labelsOf(repos []Repo) []string {
	out := make([]string, len(repos))
	for i, r := range repos {
		out[i] = r.Label
	}
	return out
}

func assertLabels(t *testing.T, got []Repo, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("labels = %v, want %v", labelsOf(got), want)
	}
	for i, r := range got {
		if r.Label != want[i] {
			t.Errorf("labels = %v, want %v", labelsOf(got), want)
			return
		}
	}
}

func TestRepos(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, root, outside string)
		want  []string
	}{
		{
			name: "flat folder of repositories",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "web"))
				mkRepo(t, filepath.Join(root, "api"))
				mkDir(t, filepath.Join(root, "notes"))
			},
			want: []string{"api", "web"},
		},
		{
			name: "nested group",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "group", "api"))
			},
			want: []string{filepath.Join("group", "api")},
		},
		{
			name: "discovery stops at the repository root",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "api"))
				mkRepo(t, filepath.Join(root, "api", "packages", "core"))
			},
			want: []string{"api"},
		},
		{
			name: "deeply nested repository is found",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "a", "b", "c", "d", "e", "repo"))
			},
			want: []string{filepath.Join("a", "b", "c", "d", "e", "repo")},
		},
		{
			name: "symlink to an external repository keeps the link name",
			setup: func(t *testing.T, root, outside string) {
				mkRepo(t, filepath.Join(outside, "shared-lib"))
				symlink(t, filepath.Join(outside, "shared-lib"), filepath.Join(root, "shared"))
			},
			want: []string{"shared"},
		},
		{
			name: "link to the folder itself does not loop",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "api"))
				symlink(t, root, filepath.Join(root, "loop"))
			},
			want: []string{"api"},
		},
		{
			name: "link to an ancestor does not loop nor leak siblings",
			setup: func(t *testing.T, root, outside string) {
				mkRepo(t, filepath.Join(root, "api"))
				mkRepo(t, filepath.Join(outside, "sibling"))
				symlink(t, filepath.Dir(root), filepath.Join(root, "up"))
			},
			want: []string{"api"},
		},
		{
			name: "same repository via link and real path is reported once",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "api"))
				symlink(t, filepath.Join(root, "api"), filepath.Join(root, "api-link"))
			},
			want: []string{"api"},
		},
		{
			name: "dangling link is skipped",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "api"))
				symlink(t, filepath.Join(root, "does-not-exist"), filepath.Join(root, "broken"))
			},
			want: []string{"api"},
		},
		{
			name: "node_modules and hidden directories are ignored",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "api"))
				mkRepo(t, filepath.Join(root, "node_modules", "pkg"))
				mkRepo(t, filepath.Join(root, ".cache", "tool"))
			},
			want: []string{"api"},
		},
		{
			name: "symlink to a file is not a repository",
			setup: func(t *testing.T, root, _ string) {
				mkRepo(t, filepath.Join(root, "api"))
				symlink(t, filepath.Join(root, "api", "package.json"), filepath.Join(root, "manifest-link"))
			},
			want: []string{"api"},
		},
		{
			name:  "empty folder yields nothing",
			setup: func(t *testing.T, root, _ string) {},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: root and outside share a parent so ancestor links are testable
			parent := t.TempDir()
			root := filepath.Join(parent, "hub")
			outside := filepath.Join(parent, "outside")
			mkDir(t, root)
			mkDir(t, outside)
			tt.setup(t, root, outside)

			// Act
			got, err := Repos(root)

			// Assert
			if err != nil {
				t.Fatalf("Repos() error = %v", err)
			}
			assertLabels(t, got, tt.want...)
		})
	}
}

func TestReposDirIsThePathAsReached(t *testing.T) {
	// Arrange
	parent := t.TempDir()
	root := filepath.Join(parent, "hub")
	mkDir(t, root)
	mkRepo(t, filepath.Join(parent, "shared-lib"))
	symlink(t, filepath.Join(parent, "shared-lib"), filepath.Join(root, "shared"))

	// Act
	got, err := Repos(root)

	// Assert
	if err != nil || len(got) != 1 {
		t.Fatalf("Repos() = %v, %v; want one repo", got, err)
	}
	if want := filepath.Join(root, "shared"); got[0].Dir != want {
		t.Errorf("Dir = %q, want the link path %q", got[0].Dir, want)
	}
}

func TestReposUnreadableSubdirectoryIsSkipped(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	// Arrange
	root := t.TempDir()
	mkRepo(t, filepath.Join(root, "api"))
	locked := filepath.Join(root, "locked")
	mkRepo(t, filepath.Join(locked, "hidden-repo"))
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	// Act
	got, err := Repos(root)

	// Assert
	if err != nil {
		t.Fatalf("Repos() error = %v, want unreadable subtree skipped", err)
	}
	assertLabels(t, got, "api")
}

func TestReposUnreadableRootIsAnError(t *testing.T) {
	// Arrange
	root := filepath.Join(t.TempDir(), "missing")

	// Act
	_, err := Repos(root)

	// Assert
	if err == nil {
		t.Fatal("Repos() on a missing root must fail")
	}
}
