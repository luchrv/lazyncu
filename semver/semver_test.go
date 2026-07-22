package semver

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name    string
		current string
		next    string
		want    Severity
	}{
		{"major jump", "2.4.1", "3.0.0", Major},
		{"minor jump", "2.4.1", "2.5.0", Minor},
		{"patch jump", "2.4.1", "2.4.9", Patch},
		{"caret prefix stripped", "^2.4.1", "^3.0.0", Major},
		{"tilde prefix stripped", "~1.2.3", "~1.3.0", Minor},
		{"gte prefix stripped", ">=1.2.3", ">=1.2.4", Patch},
		{"mixed prefixes", "^2.4.1", "2.5.0", Minor},
		{"prerelease to release major", "1.0.0-beta.1", "2.0.0", Major},
		{"git url is other", "git+https://github.com/u/r.git", "3.0.0", Other},
		{"dist tag is other", "latest", "3.0.0", Other},
		{"wildcard is other", "*", "3.0.0", Other},
		{"unknown current is other", "", "3.0.0", Other},
		{"unparseable next is other", "1.2.3", "not-a-version", Other},
		{"equal versions is other", "1.2.3", "1.2.3", Other},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := Classify(tt.current, tt.next)

			// Assert
			if got != tt.want {
				t.Errorf("Classify(%q, %q) = %v, want %v", tt.current, tt.next, got, tt.want)
			}
		})
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name       string
		severities []Severity
		want       Counters
	}{
		{
			name:       "mixed severities",
			severities: []Severity{Major, Major, Major, Minor, Minor, Minor, Minor, Minor, Patch, Patch},
			want:       Counters{Major: 3, Minor: 5, Patch: 2},
		},
		{
			name:       "up to date project",
			severities: nil,
			want:       Counters{},
		},
		{
			name:       "other excluded from counters",
			severities: []Severity{Major, Other, Other},
			want:       Counters{Major: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := Count(tt.severities)

			// Assert
			if got != tt.want {
				t.Errorf("Count() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCountersTotal(t *testing.T) {
	// Arrange
	c := Counters{Major: 1, Minor: 2, Patch: 3}

	// Act & Assert
	if got := c.Total(); got != 6 {
		t.Errorf("Total() = %d, want 6", got)
	}
}
