package audit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/luchrv/lazyncu/detect"
)

// fakeRunner mirrors the scanner test double: canned responses per command.
type fakeRunner struct {
	responses map[string]fakeResponse
	calls     []string
}

type fakeResponse struct {
	stdout []byte
	err    error
}

func (f *fakeRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	key := strings.Join(append([]string{name}, args...), " ")
	f.calls = append(f.calls, key)
	resp, ok := f.responses[key]
	if !ok {
		return nil, errors.New("fakeRunner: unexpected command " + key)
	}
	return resp.stdout, resp.err
}

const auditWithVulns = `{
  "auditReportVersion": 2,
  "vulnerabilities": {
    "lodash": {
      "name": "lodash", "severity": "critical", "isDirect": false,
      "via": [{"source": 1523, "title": "Prototype Pollution"}],
      "effects": ["express"], "range": "<4.17.21",
      "fixAvailable": true
    },
    "express": {
      "name": "express", "severity": "high", "isDirect": true,
      "via": ["lodash"], "effects": [], "range": "<4.19.0",
      "fixAvailable": {"name": "express", "version": "4.19.2", "isSemVerMajor": false}
    },
    "minimist": {
      "name": "minimist", "severity": "moderate", "isDirect": true,
      "via": [{"source": 88, "title": "Prototype Pollution"}],
      "effects": [], "range": "<1.2.6",
      "fixAvailable": false
    },
    "tmp": {
      "name": "tmp", "severity": "low", "isDirect": true,
      "via": [{"source": 99, "title": "Symlink"}],
      "effects": [], "range": "<0.2.4",
      "fixAvailable": true
    }
  },
  "metadata": {"vulnerabilities": {"info":0,"low":1,"moderate":1,"high":1,"critical":1,"total":4}}
}`

const auditClean = `{
  "auditReportVersion": 2,
  "vulnerabilities": {},
  "metadata": {"vulnerabilities": {"info":0,"low":0,"moderate":0,"high":0,"critical":0,"total":0}}
}`

func vulnByName(t *testing.T, res Result, name string) Vulnerability {
	t.Helper()
	for _, v := range res.Vulns {
		if v.Name == name {
			return v
		}
	}
	t.Fatalf("vulnerability %q not found in %+v", name, res.Vulns)
	return Vulnerability{}
}

func TestRunNpmParsesCountersAndDetail(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm audit --json": {stdout: []byte(auditWithVulns), err: errors.New("exit status 1")},
	}}

	// Act
	res := Run(context.Background(), r, "/p/api", detect.Npm)

	// Assert
	if res.Status != StatusOK {
		t.Fatalf("Status = %v, want StatusOK (exit 1 with valid JSON is success)", res.Status)
	}
	want := Counters{Critical: 1, High: 1, Moderate: 1, Low: 1}
	if res.Counters != want {
		t.Errorf("Counters = %+v, want %+v", res.Counters, want)
	}
	lodash := vulnByName(t, res, "lodash")
	if lodash.Severity != Critical || lodash.Range != "<4.17.21" || !lodash.FixAvailable {
		t.Errorf("lodash = %+v, want critical, <4.17.21, fix available", lodash)
	}
	minimist := vulnByName(t, res, "minimist")
	if minimist.FixAvailable {
		t.Error("minimist.FixAvailable = true, want false")
	}
	express := vulnByName(t, res, "express")
	if !express.FixAvailable {
		t.Error("express.FixAvailable = false, want true (object form counts as available)")
	}
}

func TestRunPnpmUsesPnpmAudit(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"pnpm audit --json": {stdout: []byte(auditClean)},
	}}

	// Act
	res := Run(context.Background(), r, "/p/web", detect.Pnpm)

	// Assert
	if res.Status != StatusOK {
		t.Fatalf("Status = %v, want StatusOK", res.Status)
	}
	if res.Counters != (Counters{}) || len(res.Vulns) != 0 {
		t.Errorf("clean audit = %+v, want zero vulnerabilities", res)
	}
}

func TestRunZeroVulnsIsOKNotUnavailable(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm audit --json": {stdout: []byte(auditClean)},
	}}

	// Act
	res := Run(context.Background(), r, "/p/api", detect.Npm)

	// Assert
	if res.Status != StatusOK {
		t.Errorf("Status = %v, want StatusOK — '0 vulnerabilities' must be distinct from 'not available'", res.Status)
	}
}

func TestRunYarnIsNotAvailableAndRunsNothing(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{}}

	// Act
	res := Run(context.Background(), r, "/p/cli", detect.Yarn)

	// Assert
	if res.Status != StatusNotAvailable {
		t.Errorf("Status = %v, want StatusNotAvailable for yarn", res.Status)
	}
	if len(r.calls) != 0 {
		t.Errorf("runner was called %v; yarn projects must not execute any audit command", r.calls)
	}
}

func TestGlobalResultIsNotAvailable(t *testing.T) {
	// Act & Assert
	if got := GlobalResult(); got.Status != StatusNotAvailable {
		t.Errorf("GlobalResult().Status = %v, want StatusNotAvailable", got.Status)
	}
}

func TestRunExecFailureIsFailed(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm audit --json": {err: errors.New("spawn failed")},
	}}

	// Act
	res := Run(context.Background(), r, "/p/api", detect.Npm)

	// Assert
	if res.Status != StatusFailed {
		t.Errorf("Status = %v, want StatusFailed", res.Status)
	}
	if res.Err == "" {
		t.Error("Err is empty, want failure reason for the UI")
	}
}

func TestRunMalformedJSONIsFailed(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm audit --json": {stdout: []byte("not json"), err: errors.New("exit status 1")},
	}}

	// Act
	res := Run(context.Background(), r, "/p/api", detect.Npm)

	// Assert
	if res.Status != StatusFailed {
		t.Errorf("Status = %v, want StatusFailed for unparseable output", res.Status)
	}
}

// --- Dependency chain ---

func TestChainTransitive(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm audit --json": {stdout: []byte(auditWithVulns)},
	}}

	// Act
	res := Run(context.Background(), r, "/p/api", detect.Npm)

	// Assert
	lodash := vulnByName(t, res, "lodash")
	if lodash.Direct {
		t.Error("lodash.Direct = true, want false")
	}
	if len(lodash.Chain) != 2 || lodash.Chain[0] != "lodash" || lodash.Chain[1] != "express" {
		t.Errorf("lodash.Chain = %v, want [lodash express]", lodash.Chain)
	}
}

func TestChainDirectDependency(t *testing.T) {
	// Arrange
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm audit --json": {stdout: []byte(auditWithVulns)},
	}}

	// Act
	res := Run(context.Background(), r, "/p/api", detect.Npm)

	// Assert
	express := vulnByName(t, res, "express")
	if !express.Direct {
		t.Error("express.Direct = false, want true")
	}
	if len(express.Chain) != 0 {
		t.Errorf("express.Chain = %v, want empty for direct dependency", express.Chain)
	}
}

func TestChainCycleSafety(t *testing.T) {
	// Arrange: a ← b ← a cycle in effects
	cyclic := `{
	  "auditReportVersion": 2,
	  "vulnerabilities": {
	    "a": {"name":"a","severity":"high","isDirect":false,"via":[{"source":1}],"effects":["b"],"range":"*","fixAvailable":false},
	    "b": {"name":"b","severity":"high","isDirect":false,"via":["a"],"effects":["a"],"range":"*","fixAvailable":false}
	  },
	  "metadata": {"vulnerabilities":{"info":0,"low":0,"moderate":0,"high":2,"critical":0,"total":2}}
	}`
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm audit --json": {stdout: []byte(cyclic)},
	}}

	// Act: must terminate
	res := Run(context.Background(), r, "/p/api", detect.Npm)

	// Assert
	if res.Status != StatusOK {
		t.Fatalf("Status = %v, want StatusOK", res.Status)
	}
	a := vulnByName(t, res, "a")
	if len(a.Chain) > MaxChainHops+1 {
		t.Errorf("cyclic chain %v exceeds bound %d", a.Chain, MaxChainHops+1)
	}
}

func TestChainTruncationPastMaxHops(t *testing.T) {
	// Arrange: v ← e1 ← e2 ← e3 ← e4 ← e5 (deeper than MaxChainHops)
	deep := `{
	  "auditReportVersion": 2,
	  "vulnerabilities": {
	    "v":  {"name":"v","severity":"low","isDirect":false,"via":[{"source":1}],"effects":["e1"],"range":"*","fixAvailable":false},
	    "e1": {"name":"e1","severity":"low","isDirect":false,"via":["v"],"effects":["e2"],"range":"*","fixAvailable":false},
	    "e2": {"name":"e2","severity":"low","isDirect":false,"via":["e1"],"effects":["e3"],"range":"*","fixAvailable":false},
	    "e3": {"name":"e3","severity":"low","isDirect":false,"via":["e2"],"effects":["e4"],"range":"*","fixAvailable":false},
	    "e4": {"name":"e4","severity":"low","isDirect":false,"via":["e3"],"effects":["e5"],"range":"*","fixAvailable":false},
	    "e5": {"name":"e5","severity":"low","isDirect":true,"via":["e4"],"effects":[],"range":"*","fixAvailable":false}
	  },
	  "metadata": {"vulnerabilities":{"info":0,"low":6,"moderate":0,"high":0,"critical":0,"total":6}}
	}`
	r := &fakeRunner{responses: map[string]fakeResponse{
		"npm audit --json": {stdout: []byte(deep)},
	}}

	// Act
	res := Run(context.Background(), r, "/p/api", detect.Npm)

	// Assert
	v := vulnByName(t, res, "v")
	last := v.Chain[len(v.Chain)-1]
	if last != TruncationMarker {
		t.Errorf("deep chain %v must end with truncation marker %q", v.Chain, TruncationMarker)
	}
	if len(v.Chain) != MaxChainHops+1 {
		t.Errorf("truncated chain length = %d, want %d", len(v.Chain), MaxChainHops+1)
	}
}

// --- Fix commands ---

func TestFixCommand(t *testing.T) {
	tests := []struct {
		name string
		res  Result
		dir  string
		pm   detect.PackageManager
		want string
	}{
		{
			name: "npm with vulns",
			res:  Result{Status: StatusOK, Counters: Counters{High: 1}},
			dir:  "/p/api", pm: detect.Npm,
			want: "cd /p/api && npm audit fix",
		},
		{
			name: "pnpm with vulns",
			res:  Result{Status: StatusOK, Counters: Counters{Critical: 2}},
			dir:  "/p/web", pm: detect.Pnpm,
			want: "cd /p/web && pnpm audit --fix",
		},
		{
			name: "zero vulnerabilities means no command",
			res:  Result{Status: StatusOK},
			dir:  "/p/api", pm: detect.Npm,
			want: "",
		},
		{
			name: "not available means no command",
			res:  Result{Status: StatusNotAvailable},
			dir:  "/p/cli", pm: detect.Yarn,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := FixCommand(tt.res, tt.dir, tt.pm)

			// Assert
			if got != tt.want {
				t.Errorf("FixCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}
