package registry

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeRunner replies with canned output keyed by "name arg1 arg2 ...".
type fakeRunner struct {
	responses map[string][]byte
	errs      map[string]error
	calls     []string
}

func (f *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := name
	for _, a := range args {
		key += " " + a
	}
	f.calls = append(f.calls, key)
	if err, ok := f.errs[key]; ok {
		return nil, err
	}
	if out, ok := f.responses[key]; ok {
		return out, nil
	}
	return nil, errors.New("fakeRunner: unexpected command " + key)
}

func TestFetchParsesArrayOutput(t *testing.T) {
	// Arrange: npm 12 wraps results in an array with package fields repeated
	out := []byte(`[
		{"version":"4.17.20","engines":{"node":">=4"},"dist-tags":{"latest":"4.17.21"},"time":{"4.17.20":"2020-07-06T00:00:00Z"}},
		{"version":"4.17.21","engines":{"node":">=4"},"dist-tags":{"latest":"4.17.21"},"time":{"4.17.21":"2021-02-20T00:00:00Z"}}
	]`)
	r := &fakeRunner{responses: map[string][]byte{
		"npm view lodash@>=0.0.0-0 version engines dist-tags deprecated --json": out,
	}}

	// Act
	md, err := NewFetcher(r).Fetch(context.Background(), "", "lodash")

	// Assert
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if md.DistTags["latest"] != "4.17.21" {
		t.Errorf("dist-tags = %v, want latest 4.17.21", md.DistTags)
	}
	if len(md.Versions) != 2 || md.Versions[1].Version != "4.17.21" || md.Versions[1].EnginesNode != ">=4" {
		t.Errorf("versions = %+v, want two versions with engines", md.Versions)
	}
	if md.Versions[1].Time != "2021-02-20T00:00:00Z" {
		t.Errorf("time = %q, want the publish time", md.Versions[1].Time)
	}
}

func TestFetchToleratesArrayFields(t *testing.T) {
	// Arrange: npm renders absent fields as empty arrays, not omissions
	out := []byte(`[{"version":"1.0.0","engines":[],"dist-tags":{"latest":"1.0.0"},"deprecated":[],"time":{}}]`)
	r := &fakeRunner{responses: map[string][]byte{
		"npm view x@>=0.0.0-0 version engines dist-tags deprecated --json": out,
	}}

	// Act
	md, err := NewFetcher(r).Fetch(context.Background(), "", "x")

	// Assert
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(md.Versions) != 1 || md.Versions[0].EnginesNode != "" || md.Versions[0].Deprecated {
		t.Errorf("versions = %+v, want one version with empty engines and not deprecated", md.Versions)
	}
}

func TestFetchParsesSingleObjectOutput(t *testing.T) {
	// Arrange: a single version may come back as a bare object
	out := []byte(`{"version":"2.0.0","engines":{"node":">=18"},"dist-tags":{"latest":"2.0.0"}}`)
	r := &fakeRunner{responses: map[string][]byte{
		"npm view x@>=0.0.0-0 version engines dist-tags deprecated --json": out,
	}}

	// Act
	md, err := NewFetcher(r).Fetch(context.Background(), "", "x")

	// Assert
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(md.Versions) != 1 || md.Versions[0].Version != "2.0.0" {
		t.Errorf("versions = %+v, want one version 2.0.0", md.Versions)
	}
}

func TestFetchPropagatesError(t *testing.T) {
	r := &fakeRunner{errs: map[string]error{
		"npm view x@>=0.0.0-0 version engines dist-tags deprecated --json": errors.New("exit status 1"),
	}}
	if _, err := NewFetcher(r).Fetch(context.Background(), "", "x"); err == nil {
		t.Fatal("Fetch() = nil, want error when npm view fails")
	}
}

func TestFetchAcceptsCollapsedVersionStrings(t *testing.T) {
	// Arrange: when no entry has a field besides version, npm collapses
	// each entry to the bare version string
	r := &fakeRunner{responses: map[string][]byte{
		"npm view x@>=0.0.0-0 version engines dist-tags deprecated --json": []byte(`["1.0.0","1.1.0"]`),
	}}

	// Act
	md, err := NewFetcher(r).Fetch(context.Background(), "", "x")

	// Assert
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(md.Versions) != 2 || md.Versions[1].Version != "1.1.0" {
		t.Errorf("versions = %+v, want 1.0.0 and 1.1.0", md.Versions)
	}
}

func TestPublishTimesParsesShapes(t *testing.T) {
	cases := map[string]string{
		"bare map":    `{"created":"2020-01-01T00:00:00Z","1.0.0":"2020-01-02T00:00:00Z"}`,
		"wrapped map": `[{"created":"2020-01-01T00:00:00Z","1.0.0":"2020-01-02T00:00:00Z"}]`,
		"keyed map":   `[{"time":{"1.0.0":"2020-01-02T00:00:00Z"}}]`,
	}
	for name, out := range cases {
		t.Run(name, func(t *testing.T) {
			r := &fakeRunner{responses: map[string][]byte{"npm view x time --json": []byte(out)}}

			times, err := NewFetcher(r).PublishTimes(context.Background(), "", "x")

			if err != nil {
				t.Fatalf("PublishTimes() error = %v", err)
			}
			if times["1.0.0"] != "2020-01-02T00:00:00Z" {
				t.Errorf("times = %v, want 1.0.0 publish time", times)
			}
		})
	}
}

func TestPublishTimesPropagatesError(t *testing.T) {
	r := &fakeRunner{errs: map[string]error{"npm view x time --json": errors.New("exit status 1")}}
	if _, err := NewFetcher(r).PublishTimes(context.Background(), "", "x"); err == nil {
		t.Fatal("PublishTimes() = nil, want error when npm view fails")
	}
}

func TestParseEmptyResponse(t *testing.T) {
	if _, err := parse([]byte("  ")); err == nil {
		t.Fatal("parse(empty) = nil, want error")
	}
}

func versions(list ...Version) []Version { return list }

func TestTargetLatestTagUsed(t *testing.T) {
	// Arrange
	md := Metadata{
		DistTags: map[string]string{"latest": "1.1.0"},
		Versions: versions(
			Version{Version: "1.0.0"},
			Version{Version: "1.1.0"},
		),
	}

	// Act
	got, ok := Target(md, "^1.0.0", Options{AllowDeprecated: true})

	// Assert
	if !ok || got != "1.1.0" {
		t.Errorf("Target() = %q, %v, want 1.1.0, true", got, ok)
	}
}

func TestTargetLatestPrereleaseFallsBackToGreatestStable(t *testing.T) {
	// Arrange: latest is a prerelease, current is stable
	md := Metadata{
		DistTags: map[string]string{"latest": "2.0.0-beta.1"},
		Versions: versions(
			Version{Version: "1.0.0"},
			Version{Version: "1.5.0"},
			Version{Version: "2.0.0-beta.1"},
		),
	}

	// Act
	got, ok := Target(md, "^1.0.0", Options{AllowDeprecated: true})

	// Assert
	if !ok || got != "1.5.0" {
		t.Errorf("Target() = %q, %v, want 1.5.0, true", got, ok)
	}
}

func TestTargetPrereleaseCurrentAllowsPre(t *testing.T) {
	// Arrange
	md := Metadata{
		DistTags: map[string]string{"latest": "2.0.0-beta.5"},
		Versions: versions(
			Version{Version: "2.0.0-beta.1"},
			Version{Version: "2.0.0-beta.5"},
		),
	}

	// Act: the caller mirrors ncu by allowing pre for a prerelease current
	got, ok := Target(md, "2.0.0-beta.1", Options{AllowDeprecated: true, AllowPre: IsPre("2.0.0-beta.1")})

	// Assert
	if !ok || got != "2.0.0-beta.5" {
		t.Errorf("Target() = %q, %v, want 2.0.0-beta.5, true", got, ok)
	}
}

func TestTargetEnginesIncompatibleFallsBack(t *testing.T) {
	// Arrange: latest needs node >=20, project is >=16 <17
	md := Metadata{
		DistTags: map[string]string{"latest": "3.0.0"},
		Versions: versions(
			Version{Version: "3.0.0", EnginesNode: ">=20"},
			Version{Version: "2.0.0", EnginesNode: ">=16"},
		),
	}

	// Act
	got, ok := Target(md, "^1.0.0", Options{EnginesNode: ">=16 <17", AllowDeprecated: true})

	// Assert
	if !ok || got != "2.0.0" {
		t.Errorf("Target() = %q, %v, want 2.0.0 (newest satisfying engines)", got, ok)
	}
}

func TestTargetCandidateWithoutEnginesAllowed(t *testing.T) {
	md := Metadata{
		DistTags: map[string]string{"latest": "2.0.0"},
		Versions: versions(Version{Version: "2.0.0"}),
	}
	got, ok := Target(md, "^1.0.0", Options{EnginesNode: ">=16 <17", AllowDeprecated: true})
	if !ok || got != "2.0.0" {
		t.Errorf("Target() = %q, %v, want 2.0.0 allowed (candidate declares no engines)", got, ok)
	}
}

func TestTargetDeprecatedAllowedByDefault(t *testing.T) {
	md := Metadata{
		DistTags: map[string]string{"latest": "2.0.0"},
		Versions: versions(Version{Version: "2.0.0", Deprecated: true}),
	}
	got, ok := Target(md, "^1.0.0", Options{AllowDeprecated: true})
	if !ok || got != "2.0.0" {
		t.Errorf("Target() = %q, %v, want deprecated 2.0.0 allowed", got, ok)
	}
}

func TestTargetCooldownExcludesFreshVersion(t *testing.T) {
	// Arrange
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	md := Metadata{
		DistTags: map[string]string{"latest": "2.0.0"},
		Versions: versions(
			Version{Version: "2.0.0", Time: now.Add(-24 * time.Hour).Format(time.RFC3339)},
			Version{Version: "1.5.0", Time: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)},
		),
	}

	// Act: 7-day cooldown
	got, ok := Target(md, "^1.0.0", Options{AllowDeprecated: true, CooldownDays: 7, Now: now})

	// Assert
	if !ok || got != "1.5.0" {
		t.Errorf("Target() = %q, %v, want 1.5.0 (2.0.0 too fresh)", got, ok)
	}
}

func TestTargetNoUpgradeWhenNotNewer(t *testing.T) {
	md := Metadata{
		DistTags: map[string]string{"latest": "1.0.0"},
		Versions: versions(Version{Version: "1.0.0"}),
	}
	if got, ok := Target(md, "^1.0.0", Options{AllowDeprecated: true}); ok {
		t.Errorf("Target() = %q, %v, want no upgrade", got, ok)
	}
}

func TestTargetUnknownCurrentSpec(t *testing.T) {
	md := Metadata{
		DistTags: map[string]string{"latest": "1.0.0"},
		Versions: versions(Version{Version: "1.0.0"}),
	}
	if _, ok := Target(md, "latest", Options{}); ok {
		t.Error("Target() ok = true for a dist-tag current spec, want false")
	}
}

func TestIsRegistrySpec(t *testing.T) {
	tests := []struct {
		spec string
		want bool
	}{
		{"^1.2.3", true},
		{"~1.2.3", true},
		{"*", true},
		{"@types/node", true},
		{"file:../local", false},
		{"link:../local", false},
		{"workspace:*", false},
		{"catalog:default", false},
		{"portal:../x", false},
		{"npm:other@1.0.0", false},
		{"git+https://github.com/a/b.git", false},
		{"github:user/repo", false},
		{"raineorshine/foo", false},
	}
	for _, tt := range tests {
		if got := IsRegistrySpec(tt.spec); got != tt.want {
			t.Errorf("IsRegistrySpec(%q) = %v, want %v", tt.spec, got, tt.want)
		}
	}
}
