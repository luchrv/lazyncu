package registry

import (
	"errors"
	"regexp"
	"strings"
	"time"

	masterminds "github.com/Masterminds/semver/v3"
)

// Options controls version selection for one project.
type Options struct {
	// EnginesNode is the project's declared engines.node constraint (empty
	// disables the engines filter).
	EnginesNode string
	// AllowPre includes prereleases regardless of the current version.
	AllowPre bool
	// AllowDeprecated includes deprecated versions (npm-check-updates'
	// default).
	AllowDeprecated bool
	// DistTag selects the target tag; empty means "latest".
	DistTag string
	// CooldownDays excludes versions published more recently than this.
	CooldownDays int
	// Now is the clock for cooldown checks; zero means time.Now().
	Now time.Time
}

// Target returns the version to suggest for a dependency declared as
// currentSpec, and whether it is an upgrade (strictly newer than the current
// minimum). A non-registry or unparseable spec yields ("", false).
func Target(md Metadata, currentSpec string, opts Options) (string, bool) {
	cur, err := MinVersion(currentSpec)
	if err != nil {
		return "", false
	}
	target := chooseTarget(md, opts)
	if target == "" {
		return "", false
	}
	tv, err := masterminds.NewVersion(target)
	if err != nil || !tv.GreaterThan(cur) {
		return "", false
	}
	return target, true
}

// chooseTarget prefers the version on the target dist-tag when it passes the
// filters; otherwise it falls back to the greatest version that passes.
func chooseTarget(md Metadata, opts Options) string {
	tag := opts.DistTag
	if tag == "" {
		tag = "latest"
	}
	allowPre := opts.AllowPre || (opts.DistTag != "" && opts.DistTag != "latest")
	tagVersion := md.DistTags[tag]

	best := ""
	var bestV *masterminds.Version
	for _, v := range md.Versions {
		if !passes(v, opts, allowPre) {
			continue
		}
		ver, err := masterminds.NewVersion(v.Version)
		if err != nil {
			continue
		}
		if bestV == nil || ver.GreaterThan(bestV) {
			bestV, best = ver, v.Version
		}
	}
	if tagVersion != "" {
		for _, v := range md.Versions {
			if v.Version == tagVersion && passes(v, opts, allowPre) {
				return tagVersion
			}
		}
	}
	return best
}

// passes applies the prerelease, deprecated, engines and cooldown filters to
// one published version.
func passes(v Version, opts Options, allowPre bool) bool {
	ver, err := masterminds.NewVersion(v.Version)
	if err != nil {
		return false
	}
	if !allowPre && ver.Prerelease() != "" {
		return false
	}
	if !opts.AllowDeprecated && v.Deprecated {
		return false
	}
	if opts.EnginesNode != "" && v.EnginesNode != "" && !enginesCompatible(opts.EnginesNode, v.EnginesNode) {
		return false
	}
	if opts.CooldownDays > 0 && v.Time != "" {
		if published, err := time.Parse(time.RFC3339, v.Time); err == nil {
			now := opts.Now
			if now.IsZero() {
				now = time.Now()
			}
			if now.Sub(published) < time.Duration(opts.CooldownDays)*24*time.Hour {
				return false
			}
		}
	}
	return true
}

// enginesCompatible reports whether the project's minimum node version
// satisfies the candidate version's engines.node. An unparseable constraint on
// either side is treated as compatible (never over-filter).
func enginesCompatible(projectEngines, candidateEngines string) bool {
	min, err := MinVersion(projectEngines)
	if err != nil {
		return true
	}
	constraint, err := masterminds.NewConstraint(candidateEngines)
	if err != nil {
		return true
	}
	return constraint.Check(min)
}

// IsPre reports whether a version declaration is a prerelease.
func IsPre(spec string) bool {
	v, err := MinVersion(spec)
	return err == nil && v.Prerelease() != ""
}

// versionToken matches a version-like substring inside a range expression.
var versionToken = regexp.MustCompile(`\d+(\.\d+){0,2}(-[0-9A-Za-z.]+)?`)

// MinVersion extracts the minimum concrete version from a range declaration
// (e.g. ">=16 <17" -> 16.0.0, "^2.3.4" -> 2.3.4). It errors when no version is
// present (dist-tags, git URLs, or malformed specs).
func MinVersion(spec string) (*masterminds.Version, error) {
	trimmed := strings.TrimSpace(spec)
	if v, err := masterminds.NewVersion(trimmed); err == nil {
		return v, nil
	}
	var min *masterminds.Version
	for _, tok := range versionToken.FindAllString(trimmed, -1) {
		v, err := masterminds.NewVersion(tok)
		if err != nil {
			continue
		}
		if min == nil || v.LessThan(min) {
			min = v
		}
	}
	if min == nil {
		return nil, errors.New("no version in spec")
	}
	return min, nil
}

// IsRegistrySpec reports whether a dependency declaration refers to a version
// on a registry (as opposed to a local path, protocol, git URL or alias that
// npm-check-updates ignores).
func IsRegistrySpec(spec string) bool {
	s := strings.TrimSpace(spec)
	for _, prefix := range []string{"file:", "link:", "workspace:", "catalog:", "portal:", "npm:", "git+", "git://", "github:", "http:", "https:"} {
		if strings.HasPrefix(s, prefix) {
			return false
		}
	}
	if strings.Contains(s, "://") {
		return false
	}
	// short GitHub references like "owner/repo" (scoped packages start with @)
	if !strings.HasPrefix(s, "@") && strings.Contains(s, "/") {
		return false
	}
	return true
}
