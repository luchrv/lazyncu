package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// AbbreviatedAccept requests the abbreviated packument: versions with
// engines and deprecation plus dist-tags, a fraction of the full document.
const AbbreviatedAccept = "application/vnd.npm.install-v1+json"

// parsePackument decodes a registry packument (abbreviated or full) into
// Metadata. Versions are returned in ascending publish order when `time` is
// present, otherwise in document order.
func parsePackument(data []byte) (Metadata, error) {
	var doc struct {
		DistTags map[string]string          `json:"dist-tags"`
		Versions map[string]json.RawMessage `json:"versions"`
		Time     map[string]string          `json:"time"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return Metadata{}, fmt.Errorf("parsing packument: %w", err)
	}
	if len(doc.Versions) == 0 {
		return Metadata{}, errors.New("packument has no versions")
	}
	md := Metadata{DistTags: doc.DistTags, Versions: make([]Version, 0, len(doc.Versions))}
	if md.DistTags == nil {
		md.DistTags = map[string]string{}
	}
	for version, raw := range doc.Versions {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return Metadata{}, fmt.Errorf("parsing packument version %s: %w", version, err)
		}
		md.Versions = append(md.Versions, Version{
			Version:     version,
			EnginesNode: enginesNode(fields["engines"]),
			Deprecated:  isDeprecated(fields["deprecated"]),
			Time:        doc.Time[version],
		})
	}
	sort.Slice(md.Versions, func(i, j int) bool {
		if md.Versions[i].Time != md.Versions[j].Time {
			return md.Versions[i].Time < md.Versions[j].Time
		}
		return md.Versions[i].Version < md.Versions[j].Version
	})
	return md, nil
}

// parsePackumentTimes extracts the per-version publish times of a full
// packument.
func parsePackumentTimes(data []byte) (map[string]string, error) {
	var doc struct {
		Time map[string]string `json:"time"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing packument: %w", err)
	}
	if doc.Time == nil {
		return nil, errors.New("packument has no time field")
	}
	return doc.Time, nil
}
