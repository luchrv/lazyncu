package command

import (
	"testing"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/scanner"
)

func TestGlobalUpdate(t *testing.T) {
	tests := []struct {
		name string
		pkgs []scanner.Package
		want string
	}{
		{
			name: "multiple packages keep order",
			pkgs: []scanner.Package{
				{Name: "typescript", New: "5.6.2"},
				{Name: "eslint", New: "9.1.0"},
			},
			want: "npm install -g typescript@5.6.2 eslint@9.1.0",
		},
		{
			name: "single package",
			pkgs: []scanner.Package{{Name: "typescript", New: "5.6.2"}},
			want: "npm install -g typescript@5.6.2",
		},
		{
			name: "no updates means no command",
			pkgs: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GlobalUpdate(tt.pkgs); got != tt.want {
				t.Errorf("GlobalUpdate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectUpdate(t *testing.T) {
	lodash := scanner.Package{Name: "lodash", Current: "^4.17.20", New: "4.17.21"}
	tests := []struct {
		name string
		dir  string
		pm   detect.PackageManager
		pkgs []scanner.Package
		want string
	}{
		{"npm project", "/p/api", detect.Npm, []scanner.Package{lodash}, "cd /p/api && npm install lodash@4.17.21"},
		{"pnpm project", "/p/web", detect.Pnpm, []scanner.Package{lodash}, "cd /p/web && pnpm add lodash@4.17.21"},
		{"yarn project", "/p/cli", detect.Yarn, []scanner.Package{lodash}, "cd /p/cli && yarn add lodash@4.17.21"},
		{"unknown pm defaults to npm", "/p/x", detect.PackageManager("weird"), []scanner.Package{lodash}, "cd /p/x && npm install lodash@4.17.21"},
		{"no packages yields empty command", "/p/x", detect.Npm, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProjectUpdate(tt.dir, tt.pm, tt.pkgs); got != tt.want {
				t.Errorf("ProjectUpdate() = %q, want %q", got, tt.want)
			}
		})
	}
}
