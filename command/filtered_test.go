package command

import (
	"testing"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/scanner"
)

func TestProjectUpdateFiltered(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		pm   detect.PackageManager
		pkgs []string
		want string
	}{
		{
			name: "npm project with two packages",
			dir:  "/proj",
			pm:   detect.Npm,
			pkgs: []string{"lodash", "debug"},
			want: "cd /proj && ncu -u lodash debug && npm install",
		},
		{
			name: "pnpm project single package",
			dir:  "/proj",
			pm:   detect.Pnpm,
			pkgs: []string{"express"},
			want: "cd /proj && ncu -u express && pnpm install",
		},
		{
			name: "yarn project scoped package",
			dir:  "/proj",
			pm:   detect.Yarn,
			pkgs: []string{"@types/node"},
			want: "cd /proj && ncu -u @types/node && yarn",
		},
		{
			name: "unknown package manager defaults to npm",
			dir:  "/proj",
			pm:   detect.PackageManager("weird"),
			pkgs: []string{"lodash"},
			want: "cd /proj && ncu -u lodash && npm install",
		},
		{
			name: "no packages yields empty command",
			dir:  "/proj",
			pm:   detect.Npm,
			pkgs: nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectUpdateFiltered(tt.dir, tt.pm, tt.pkgs)
			if got != tt.want {
				t.Errorf("ProjectUpdateFiltered() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGlobalUpdateFiltered(t *testing.T) {
	all := []scanner.Package{
		{Name: "typescript", New: "5.6.2"},
		{Name: "npm-check-updates", New: "18.1.0"},
		{Name: "prettier", New: "3.9.6"},
	}

	tests := []struct {
		name   string
		marked map[string]bool
		want   string
	}{
		{
			name:   "subset of one",
			marked: map[string]bool{"typescript": true},
			want:   "npm install -g typescript@5.6.2",
		},
		{
			name:   "subset of two keeps scan order",
			marked: map[string]bool{"prettier": true, "typescript": true},
			want:   "npm install -g typescript@5.6.2 prettier@3.9.6",
		},
		{
			name:   "empty selection yields empty command",
			marked: map[string]bool{},
			want:   "",
		},
		{
			name:   "marks not present in scan are ignored",
			marked: map[string]bool{"ghost": true},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GlobalUpdateFiltered(all, tt.marked)
			if got != tt.want {
				t.Errorf("GlobalUpdateFiltered() = %q, want %q", got, tt.want)
			}
		})
	}
}
