package command

import (
	"testing"

	"github.com/luchrv/lazyncu/scanner"
)

func TestGlobalUpdateFiltered(t *testing.T) {
	all := []scanner.Package{
		{Name: "typescript", New: "5.6.2"},
		{Name: "eslint", New: "9.1.0"},
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
			if got := GlobalUpdateFiltered(all, tt.marked); got != tt.want {
				t.Errorf("GlobalUpdateFiltered() = %q, want %q", got, tt.want)
			}
		})
	}
}
