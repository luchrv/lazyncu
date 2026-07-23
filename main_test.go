package main

import "testing"

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
