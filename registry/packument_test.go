package registry

import "testing"

func TestParsePackumentAbbreviated(t *testing.T) {
	doc := []byte(`{
		"name":"lodash",
		"dist-tags":{"latest":"4.17.21"},
		"versions":{
			"4.17.20":{"version":"4.17.20","engines":{"node":">=4"},"deprecated":"old"},
			"4.17.21":{"version":"4.17.21","engines":{"node":">=4"}}
		},
		"modified":"2021-02-20T00:00:00Z"
	}`)

	md, err := parsePackument(doc)

	if err != nil {
		t.Fatalf("parsePackument() error = %v", err)
	}
	if md.DistTags["latest"] != "4.17.21" || len(md.Versions) != 2 {
		t.Fatalf("metadata = %+v, want latest 4.17.21 and two versions", md)
	}
	byVersion := map[string]Version{}
	for _, v := range md.Versions {
		byVersion[v.Version] = v
	}
	if !byVersion["4.17.20"].Deprecated || byVersion["4.17.21"].Deprecated {
		t.Errorf("deprecated flags = %+v, want only 4.17.20 deprecated", byVersion)
	}
	if byVersion["4.17.21"].EnginesNode != ">=4" {
		t.Errorf("engines = %q, want >=4", byVersion["4.17.21"].EnginesNode)
	}
}

func TestParsePackumentFullCarriesTimes(t *testing.T) {
	doc := []byte(`{
		"dist-tags":{"latest":"2.0.0"},
		"versions":{"1.0.0":{"version":"1.0.0"},"2.0.0":{"version":"2.0.0"}},
		"time":{"created":"2020-01-01T00:00:00Z","1.0.0":"2020-01-02T00:00:00Z","2.0.0":"2021-01-02T00:00:00Z"}
	}`)

	md, err := parsePackument(doc)
	times, timesErr := parsePackumentTimes(doc)

	if err != nil || timesErr != nil {
		t.Fatalf("errors = %v / %v", err, timesErr)
	}
	if md.Versions[0].Version != "1.0.0" || md.Versions[0].Time != "2020-01-02T00:00:00Z" {
		t.Errorf("versions = %+v, want ascending by publish time with times attached", md.Versions)
	}
	if times["2.0.0"] != "2021-01-02T00:00:00Z" {
		t.Errorf("times = %v", times)
	}
}

func TestParsePackumentRejectsEmptyOrInvalid(t *testing.T) {
	for name, doc := range map[string]string{
		"no versions": `{"dist-tags":{},"versions":{}}`,
		"not json":    `<html>`,
		"error body":  `{"error":"Not found"}`,
	} {
		if _, err := parsePackument([]byte(doc)); err == nil {
			t.Errorf("%s: parsePackument() = nil error, want failure", name)
		}
	}
	if _, err := parsePackumentTimes([]byte(`{"versions":{}}`)); err == nil {
		t.Error("parsePackumentTimes() without time = nil error, want failure")
	}
}
