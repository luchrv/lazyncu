package command

import (
	"testing"

	"github.com/luchrv/ncu-tui/detect"
	"github.com/luchrv/ncu-tui/scanner"
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
				{Name: "npm-check-updates", New: "18.1.0"},
			},
			want: "npm install -g typescript@5.6.2 npm-check-updates@18.1.0",
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
			// Act
			got := GlobalUpdate(tt.pkgs)

			// Assert
			if got != tt.want {
				t.Errorf("GlobalUpdate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectUpdate(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		pm   detect.PackageManager
		want string
	}{
		{"npm project", "/p/api", detect.Npm, "cd /p/api && ncu -u && npm install"},
		{"pnpm project", "/p/web", detect.Pnpm, "cd /p/web && ncu -u && pnpm install"},
		{"yarn project", "/p/cli", detect.Yarn, "cd /p/cli && ncu -u && yarn"},
		{"unknown pm defaults to npm", "/p/x", detect.PackageManager("weird"), "cd /p/x && ncu -u && npm install"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := ProjectUpdate(tt.dir, tt.pm)

			// Assert
			if got != tt.want {
				t.Errorf("ProjectUpdate() = %q, want %q", got, tt.want)
			}
		})
	}
}
