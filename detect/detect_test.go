package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanMode(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  Mode
	}{
		{
			name:  "no package.json means folder of projects",
			setup: func(t *testing.T, dir string) {},
			want:  ModeDeep,
		},
		{
			name: "package.json with workspaces field means monorepo",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"root","workspaces":["packages/*"]}`)
			},
			want: ModeDeep,
		},
		{
			name: "package.json with workspaces object means monorepo",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"root","workspaces":{"packages":["packages/*"]}}`)
			},
			want: ModeDeep,
		},
		{
			name: "package.json plus pnpm-workspace.yaml means monorepo",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"root"}`)
				writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - 'packages/*'\n")
			},
			want: ModeDeep,
		},
		{
			name: "plain package.json means single project",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"app","dependencies":{}}`)
			},
			want: ModeSingle,
		},
		{
			name: "malformed package.json still means single project",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{broken`)
			},
			want: ModeSingle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			tt.setup(t, dir)

			// Act
			got := ScanMode(dir)

			// Assert
			if got != tt.want {
				t.Errorf("ScanMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScanModeReEvaluatesEachCall(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"app"}`)
	if got := ScanMode(dir); got != ModeSingle {
		t.Fatalf("initial ScanMode() = %v, want ModeSingle", got)
	}

	// Act: the project becomes a pnpm monorepo between scans
	writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - 'apps/*'\n")

	// Assert
	if got := ScanMode(dir); got != ModeDeep {
		t.Errorf("ScanMode() after adding pnpm-workspace.yaml = %v, want ModeDeep", got)
	}
}

func TestNodeContext(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		wantNvmrc   string
		wantEngines string
	}{
		{
			name: "nvmrc content is trimmed",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, ".nvmrc", "18.19.0\n")
			},
			wantNvmrc: "18.19.0",
		},
		{
			name: "v-prefixed nvmrc is kept verbatim",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, ".nvmrc", "v20.11.1")
			},
			wantNvmrc: "v20.11.1",
		},
		{
			name: "lts alias nvmrc is kept verbatim",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, ".nvmrc", "lts/gallium\n")
			},
			wantNvmrc: "lts/gallium",
		},
		{
			name: "engines.node from package.json",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"app","engines":{"node":">=18"}}`)
			},
			wantEngines: ">=18",
		},
		{
			name: "both nvmrc and engines are reported",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, ".nvmrc", "16\n")
				writeFile(t, dir, "package.json", `{"engines":{"node":">=16 <17"}}`)
			},
			wantNvmrc:   "16",
			wantEngines: ">=16 <17",
		},
		{
			name:  "neither declaration yields empty values",
			setup: func(t *testing.T, dir string) {},
		},
		{
			name: "malformed package.json degrades to empty engines",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{broken`)
			},
		},
		{
			name: "package.json without engines yields empty engines",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "package.json", `{"name":"app","dependencies":{}}`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			tt.setup(t, dir)

			// Act
			nvmrc, engines := NodeContext(dir)

			// Assert
			if nvmrc != tt.wantNvmrc {
				t.Errorf("NodeContext() nvmrc = %q, want %q", nvmrc, tt.wantNvmrc)
			}
			if engines != tt.wantEngines {
				t.Errorf("NodeContext() engines = %q, want %q", engines, tt.wantEngines)
			}
		})
	}
}

func TestPackageManager(t *testing.T) {
	tests := []struct {
		name      string
		lockfiles []string
		want      PackageManager
	}{
		{"pnpm lockfile", []string{"pnpm-lock.yaml"}, Pnpm},
		{"yarn lockfile", []string{"yarn.lock"}, Yarn},
		{"npm lockfile", []string{"package-lock.json"}, Npm},
		{"no lockfile defaults to npm", nil, Npm},
		{"pnpm wins over yarn and npm", []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml"}, Pnpm},
		{"yarn wins over npm", []string{"package-lock.json", "yarn.lock"}, Yarn},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			for _, lf := range tt.lockfiles {
				writeFile(t, dir, lf, "")
			}

			// Act
			got := PackageManagerFor(dir)

			// Assert
			if got != tt.want {
				t.Errorf("PackageManagerFor() = %v, want %v", got, tt.want)
			}
		})
	}
}
