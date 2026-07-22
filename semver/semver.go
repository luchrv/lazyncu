// Package semver classifies package upgrades by severity (major/minor/patch)
// and aggregates per-project counters.
package semver

import (
	"strings"

	masterminds "github.com/Masterminds/semver/v3"
)

// Severity is the size of a version jump.
type Severity string

const (
	Major Severity = "major"
	Minor Severity = "minor"
	Patch Severity = "patch"
	// Other covers non-semver specs (git URLs, dist-tags, wildcards),
	// unknown current versions, and no-op jumps.
	Other Severity = "other"
)

// Counters aggregates pending upgrades per severity for one project.
// Other-severity upgrades are intentionally excluded.
type Counters struct {
	Major int
	Minor int
	Patch int
}

// Total is the number of counted pending upgrades.
func (c Counters) Total() int {
	return c.Major + c.Minor + c.Patch
}

// Classify compares current and next versions, tolerating range prefixes
// (^, ~, >=, =, v). Anything unparseable as semver classifies as Other.
func Classify(current, next string) Severity {
	cur, err := parse(current)
	if err != nil {
		return Other
	}
	nxt, err := parse(next)
	if err != nil {
		return Other
	}

	switch {
	case nxt.Major() != cur.Major():
		return Major
	case nxt.Minor() != cur.Minor():
		return Minor
	case nxt.Patch() != cur.Patch() || nxt.Prerelease() != cur.Prerelease():
		return Patch
	default:
		return Other
	}
}

// Count aggregates severities into per-project counters, excluding Other.
func Count(severities []Severity) Counters {
	var c Counters
	for _, s := range severities {
		switch s {
		case Major:
			c.Major++
		case Minor:
			c.Minor++
		case Patch:
			c.Patch++
		}
	}
	return c
}

func parse(version string) (*masterminds.Version, error) {
	trimmed := strings.TrimLeft(strings.TrimSpace(version), "^~>=< v")
	return masterminds.NewVersion(trimmed)
}
