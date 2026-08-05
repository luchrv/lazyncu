package main

import (
	"strings"
	"testing"
)

func TestWantsVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "double-dash version flag", args: []string{"lazyncu", "--version"}, want: true},
		{name: "single-dash version flag", args: []string{"lazyncu", "-version"}, want: true},
		{name: "no arguments", args: []string{"lazyncu"}, want: false},
		{name: "unrelated argument", args: []string{"lazyncu", "--help"}, want: false},
		{name: "version not first argument", args: []string{"lazyncu", "foo", "--version"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wantsVersion(tt.args)
			if got != tt.want {
				t.Errorf("wantsVersion(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

func TestPositionalPath(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantTarget string
		wantErr    bool
	}{
		{name: "no arguments", args: []string{"lazyncu"}, wantTarget: ""},
		{name: "dot argument", args: []string{"lazyncu", "."}, wantTarget: "."},
		{name: "path argument", args: []string{"lazyncu", "~/projects/api"}, wantTarget: "~/projects/api"},
		{name: "two arguments", args: []string{"lazyncu", "a", "b"}, wantErr: true},
		{name: "three arguments", args: []string{"lazyncu", "a", "b", "c"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := positionalPath(tt.args)

			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "usage: lazyncu [path]") {
					t.Errorf("positionalPath(%v) error = %v, want usage error", tt.args, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("positionalPath(%v) error: %v", tt.args, err)
			}
			if got != tt.wantTarget {
				t.Errorf("positionalPath(%v) = %q, want %q", tt.args, got, tt.wantTarget)
			}
		})
	}
}
