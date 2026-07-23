package version

import (
	"runtime/debug"
	"testing"
)

func buildInfo(mainVersion string, settings map[string]string) *debug.BuildInfo {
	bi := &debug.BuildInfo{}
	bi.Main.Version = mainVersion
	for k, v := range settings {
		bi.Settings = append(bi.Settings, debug.BuildSetting{Key: k, Value: v})
	}
	return bi
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		bi      *debug.BuildInfo
		want    Info
	}{
		{
			name:    "ldflags values win over build info",
			version: "v0.1.0",
			commit:  "abc1234",
			date:    "2026-07-22T15:04:05Z",
			bi:      buildInfo("v9.9.9", map[string]string{"vcs.revision": "ffffffffffffffffffffffffffffffffffffffff"}),
			want:    Info{Version: "v0.1.0", Commit: "abc1234", Date: "2026-07-22T15:04:05Z"},
		},
		{
			name:    "go install module version fills version",
			version: "dev",
			commit:  "none",
			date:    "unknown",
			bi:      buildInfo("v1.2.3", nil),
			want:    Info{Version: "v1.2.3", Commit: "none", Date: "unknown"},
		},
		{
			name:    "vcs settings fill commit and date on source build",
			version: "dev",
			commit:  "none",
			date:    "unknown",
			bi: buildInfo("(devel)", map[string]string{
				"vcs.revision": "0123456789abcdef0123456789abcdef01234567",
				"vcs.time":     "2026-07-22T10:00:00Z",
			}),
			want: Info{Version: "dev", Commit: "0123456", Date: "2026-07-22T10:00:00Z"},
		},
		{
			name:    "devel module version does not replace dev",
			version: "dev",
			commit:  "none",
			date:    "unknown",
			bi:      buildInfo("(devel)", nil),
			want:    Info{Version: "dev", Commit: "none", Date: "unknown"},
		},
		{
			name:    "nil build info degrades to defaults",
			version: "dev",
			commit:  "none",
			date:    "unknown",
			bi:      nil,
			want:    Info{Version: "dev", Commit: "none", Date: "unknown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolve(tt.version, tt.commit, tt.date, tt.bi)
			if got != tt.want {
				t.Errorf("resolve() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	info := Info{Version: "v0.1.0", Commit: "abc1234", Date: "2026-07-22T15:04:05Z"}

	got := info.String()

	want := "lazyncu v0.1.0 (commit abc1234, built 2026-07-22T15:04:05Z)"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestGetNeverEmpty(t *testing.T) {
	info := Get()

	if info.Version == "" || info.Commit == "" || info.Date == "" {
		t.Errorf("Get() returned empty field: %+v", info)
	}
}
