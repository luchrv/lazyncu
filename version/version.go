// Package version resolves the app's version metadata across all build
// paths: ldflags injection (make build, goreleaser) with a
// debug.ReadBuildInfo fallback (go install @version, bare source builds).
package version

import (
	"fmt"
	"runtime/debug"
)

// Defaults overridden via -ldflags "-X github.com/luchrv/lazyncu/version.<var>=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const shortCommitLen = 7

// Info holds the resolved version metadata. Fields are never empty.
type Info struct {
	Version string
	Commit  string
	Date    string
}

// String renders the single-line format used by --version and the About modal.
func (i Info) String() string {
	return fmt.Sprintf("lazyncu %s (commit %s, built %s)", i.Version, i.Commit, i.Date)
}

// Get resolves metadata from ldflags vars, falling back to build info.
func Get() Info {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		bi = nil
	}
	return resolve(version, commit, date, bi)
}

// resolve prefers ldflags-injected values; when the version is still the
// default it consults build info: the module version covers go install
// @vX.Y.Z, and vcs settings cover source builds. Missing data keeps the
// degraded defaults rather than erroring.
func resolve(v, c, d string, bi *debug.BuildInfo) Info {
	if v != "dev" || bi == nil {
		return Info{Version: v, Commit: c, Date: d}
	}
	if mv := bi.Main.Version; mv != "" && mv != "(devel)" {
		v = mv
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= shortCommitLen {
				c = s.Value[:shortCommitLen]
			} else if s.Value != "" {
				c = s.Value
			}
		case "vcs.time":
			if s.Value != "" {
				d = s.Value
			}
		}
	}
	return Info{Version: v, Commit: c, Date: d}
}
