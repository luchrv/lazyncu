// Package registry resolves the versions lazyncu suggests for upgrades by
// querying the npm registry and applying npm-check-updates' target-selection
// rules for the options lazyncu exposes. The production transport
// (HTTPFetcher) requests packuments directly using the registry, scoped
// registries, credentials, proxy and TLS settings from npm's own
// configuration, and falls back to `npm view` (NpmViewFetcher) whenever a
// request fails. Results are memoized by Cache, optionally on disk with a TTL.
package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
)

// Runner executes an external command in dir and returns its stdout (mirrors
// scanner.Runner). It is injected so tests supply canned output.
type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) ([]byte, error)
}

// Version is one published version's relevant metadata.
type Version struct {
	Version     string `json:"version"`
	EnginesNode string `json:"engines_node,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	// Time is the RFC3339 publish time; empty when unknown.
	Time string `json:"time,omitempty"`
}

// Metadata is the registry information for one package.
type Metadata struct {
	DistTags map[string]string `json:"dist_tags"`
	Versions []Version         `json:"versions"`
}

// Fetcher retrieves a package's registry metadata as seen from a project
// directory (dir selects the npm configuration in effect; "" means the
// current directory).
type Fetcher interface {
	// Fetch returns versions, per-version engines, deprecation and dist-tags.
	Fetch(ctx context.Context, dir, pkg string) (Metadata, error)
	// PublishTimes returns each version's RFC3339 publish time.
	PublishTimes(ctx context.Context, dir, pkg string) (map[string]string, error)
}

// NpmViewFetcher is the fallback Fetcher: it shells out to `npm view` in the
// project directory so that registry, scopes and credentials come from npm's
// configuration exactly as npm applies them.
type NpmViewFetcher struct{ runner Runner }

// NewFetcher builds a Fetcher around a Runner.
func NewFetcher(runner Runner) NpmViewFetcher { return NpmViewFetcher{runner: runner} }

// Fetch retrieves versions, per-version engines, dist-tags and deprecation in
// a single `npm view` call. A wide range is required because a bare `*`
// collapses to the latest version instead of enumerating all. Publish times
// are deliberately not requested here: npm repeats the package-level `time`
// map on every version entry, which makes the output quadratic in the number
// of versions (hundreds of MB for packages like @types/node). Callers that
// need them use PublishTimes.
func (f NpmViewFetcher) Fetch(ctx context.Context, dir, pkg string) (Metadata, error) {
	out, err := f.runner.Run(ctx, dir, "npm", "view", pkg+"@>=0.0.0-0",
		"version", "engines", "dist-tags", "deprecated", "--json")
	if err != nil {
		return Metadata{}, fmt.Errorf("npm view %s: %w", pkg, err)
	}
	return parse(out)
}

// PublishTimes retrieves the publish time of every version (RFC3339, keyed by
// version) with one package-level `npm view` call, whose output is linear in
// the number of versions.
func (f NpmViewFetcher) PublishTimes(ctx context.Context, dir, pkg string) (map[string]string, error) {
	out, err := f.runner.Run(ctx, dir, "npm", "view", pkg, "time", "--json")
	if err != nil {
		return nil, fmt.Errorf("npm view %s time: %w", pkg, err)
	}
	return parseTimes(out)
}

// parseTimes accepts the shapes npm uses for a single requested field: the
// bare map, the map wrapped in an array, or the map keyed under "time".
func parseTimes(out []byte) (map[string]string, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, errors.New("empty registry response")
	}
	raws := []json.RawMessage{trimmed}
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &raws); err != nil {
			return nil, fmt.Errorf("parsing registry response: %w", err)
		}
	}
	times := map[string]string{}
	for _, raw := range raws {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, fmt.Errorf("parsing registry entry: %w", err)
		}
		if nested, ok := fields["time"]; ok {
			maps.Copy(times, stringMap(nested))
			continue
		}
		for k, v := range fields {
			times[k] = stringField(v)
		}
	}
	return times, nil
}

// parse accepts both npm output shapes: a single object, or the array npm 12
// returns (the package-level fields repeat on every element). When no entry
// carries a field besides version, npm collapses each entry to the bare
// version string, which is accepted too. Individual fields are decoded
// tolerantly because npm renders an absent field as an empty array rather
// than omitting it.
func parse(out []byte) (Metadata, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return Metadata{}, errors.New("empty registry response")
	}
	raws := []json.RawMessage{trimmed}
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &raws); err != nil {
			return Metadata{}, fmt.Errorf("parsing registry response: %w", err)
		}
	}

	md := Metadata{DistTags: map[string]string{}}
	times := map[string]string{}
	seen := map[string]bool{}
	for _, raw := range raws {
		if bare := stringField(raw); bare != "" {
			raw = json.RawMessage(`{"version":` + string(raw) + `}`)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return Metadata{}, fmt.Errorf("parsing registry entry: %w", err)
		}
		for k, v := range stringMap(fields["dist-tags"]) {
			md.DistTags[k] = v
		}
		for k, v := range stringMap(fields["time"]) {
			times[k] = v
		}
		version := stringField(fields["version"])
		if version == "" || seen[version] {
			continue
		}
		seen[version] = true
		md.Versions = append(md.Versions, Version{
			Version:     version,
			EnginesNode: enginesNode(fields["engines"]),
			Deprecated:  isDeprecated(fields["deprecated"]),
			Time:        stringMap(fields["time"])[version],
		})
	}
	for i := range md.Versions {
		if md.Versions[i].Time == "" {
			md.Versions[i].Time = times[md.Versions[i].Version]
		}
	}
	if len(md.Versions) == 0 {
		return Metadata{}, errors.New("registry response has no versions")
	}
	return md, nil
}

func stringField(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return ""
}

func stringMap(raw json.RawMessage) map[string]string {
	var m map[string]string
	if json.Unmarshal(raw, &m) == nil {
		return m
	}
	return nil
}

// enginesNode decodes engines.node from either an object ({"node":"..."}) or
// the array form npm emits, taking the first entry that names a node range.
func enginesNode(raw json.RawMessage) string {
	var obj struct {
		Node string `json:"node"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Node != "" {
		return obj.Node
	}
	var arr []struct {
		Node string `json:"node"`
	}
	if json.Unmarshal(raw, &arr) == nil {
		for _, e := range arr {
			if e.Node != "" {
				return e.Node
			}
		}
	}
	return ""
}

// isDeprecated interprets npm's polymorphic field: a string message, true, or
// a non-empty array means deprecated; false, null, [] and absent mean not.
func isDeprecated(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s != ""
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		return len(arr) > 0
	}
	return true
}
