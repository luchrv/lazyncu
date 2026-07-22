// Package audit runs npm/pnpm vulnerability audits (through the injected
// scanner.Runner) and parses them into immutable results, including the
// dependency chain that drags each vulnerable package into the project.
package audit

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/luchrv/ncu-tui/detect"
	"github.com/luchrv/ncu-tui/scanner"
)

// MaxChainHops bounds the dependency chain length (package names shown);
// deeper chains are truncated with TruncationMarker.
const MaxChainHops = 4

// TruncationMarker ends a chain that was cut short (depth bound or cycle).
const TruncationMarker = "…"

// Severity is npm's audit severity scale. Info-level findings are ignored.
type Severity string

const (
	Critical Severity = "critical"
	High     Severity = "high"
	Moderate Severity = "moderate"
	Low      Severity = "low"
)

// Status distinguishes a clean audit from one that could not run: the spec
// requires "0 vulnerabilities" and "audit not available" to be distinct.
type Status string

const (
	StatusOK           Status = "ok"
	StatusNotAvailable Status = "not-available"
	StatusFailed       Status = "failed"
)

// Vulnerability is one vulnerable package in the dependency tree.
type Vulnerability struct {
	Name         string
	Severity     Severity
	Range        string
	FixAvailable bool
	// Direct marks a direct dependency of the project (no chain shown).
	Direct bool
	// Chain walks from the vulnerable package up to a direct dependency,
	// e.g. [lodash express]. Empty for direct dependencies.
	Chain []string
}

// Counters aggregates vulnerabilities per severity for one project.
type Counters struct {
	Critical int
	High     int
	Moderate int
	Low      int
}

// Total is the number of counted vulnerabilities.
func (c Counters) Total() int {
	return c.Critical + c.High + c.Moderate + c.Low
}

// Result is the audit outcome for one project.
type Result struct {
	Status   Status
	Counters Counters
	Vulns    []Vulnerability
	// Err carries the failure reason when Status is StatusFailed.
	Err string
}

// GlobalResult is the audit outcome for the global source: npm audit needs a
// lockfile and cannot audit global installs.
func GlobalResult() Result {
	return Result{Status: StatusNotAvailable}
}

// Run audits one project directory with the command matching its package
// manager. Yarn projects are not audited in v1 (different report format).
// A non-zero exit with parseable JSON is a successful audit (npm audit exits
// 1 whenever vulnerabilities exist); only unparseable output or an exec
// failure yields StatusFailed.
func Run(ctx context.Context, runner scanner.Runner, dir string, pm detect.PackageManager) Result {
	var name string
	switch pm {
	case detect.Npm:
		name = "npm"
	case detect.Pnpm:
		name = "pnpm"
	default:
		return Result{Status: StatusNotAvailable}
	}

	out, err := runner.Run(ctx, dir, name, "audit", "--json")
	if len(out) == 0 {
		reason := "audit produced no output"
		if err != nil {
			reason = err.Error()
		}
		return Result{Status: StatusFailed, Err: reason}
	}

	report, parseErr := parseReport(out)
	if parseErr != nil {
		return Result{Status: StatusFailed, Err: "unparseable audit output: " + parseErr.Error()}
	}
	return buildResult(report)
}

// FixCommand builds the copyable (never executed) fix command for a project
// with vulnerabilities; empty when there is nothing to fix or no audit ran.
func FixCommand(res Result, dir string, pm detect.PackageManager) string {
	if res.Status != StatusOK || res.Counters.Total() == 0 {
		return ""
	}
	switch pm {
	case detect.Npm:
		return "cd " + dir + " && npm audit fix"
	case detect.Pnpm:
		return "cd " + dir + " && pnpm audit --fix"
	default:
		return ""
	}
}

// reportEntry is one entry of the npm audit v2 `vulnerabilities` map.
type reportEntry struct {
	Name         string          `json:"name"`
	Severity     string          `json:"severity"`
	IsDirect     bool            `json:"isDirect"`
	Effects      []string        `json:"effects"`
	Range        string          `json:"range"`
	FixAvailable json.RawMessage `json:"fixAvailable"`
}

func parseReport(out []byte) (map[string]reportEntry, error) {
	var report struct {
		Vulnerabilities map[string]reportEntry `json:"vulnerabilities"`
	}
	if err := json.Unmarshal(out, &report); err != nil {
		return nil, err
	}
	return report.Vulnerabilities, nil
}

func buildResult(entries map[string]reportEntry) Result {
	var counters Counters
	vulns := make([]Vulnerability, 0, len(entries))

	for name, entry := range entries {
		severity, counted := countSeverity(&counters, entry.Severity)
		if !counted {
			continue
		}
		vulns = append(vulns, Vulnerability{
			Name:         name,
			Severity:     severity,
			Range:        entry.Range,
			FixAvailable: fixAvailable(entry.FixAvailable),
			Direct:       entry.IsDirect,
			Chain:        chainFor(name, entries),
		})
	}
	sort.Slice(vulns, func(i, j int) bool {
		if ri, rj := severityRank(vulns[i].Severity), severityRank(vulns[j].Severity); ri != rj {
			return ri < rj
		}
		return vulns[i].Name < vulns[j].Name
	})
	return Result{Status: StatusOK, Counters: counters, Vulns: vulns}
}

func countSeverity(c *Counters, raw string) (Severity, bool) {
	switch Severity(raw) {
	case Critical:
		c.Critical++
		return Critical, true
	case High:
		c.High++
		return High, true
	case Moderate:
		c.Moderate++
		return Moderate, true
	case Low:
		c.Low++
		return Low, true
	default: // "info" and unknown levels are not counted
		return "", false
	}
}

func severityRank(s Severity) int {
	switch s {
	case Critical:
		return 0
	case High:
		return 1
	case Moderate:
		return 2
	default:
		return 3
	}
}

// fixAvailable interprets npm's polymorphic field: false, true, or an object
// describing the fix (which counts as available).
func fixAvailable(raw json.RawMessage) bool {
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return b
	}
	return len(raw) > 0 // an object means a fix exists
}

// chainFor walks `effects` upward from a transitive vulnerable package
// toward a direct dependency: lodash → express renders as [lodash express].
// The walk is cycle-safe and bounded to MaxChainHops names; an incomplete
// walk ends with TruncationMarker. Direct dependencies have no chain.
func chainFor(name string, entries map[string]reportEntry) []string {
	entry := entries[name]
	if entry.IsDirect {
		return nil
	}

	chain := []string{name}
	visited := map[string]bool{name: true}
	current := entry

	for len(current.Effects) > 0 {
		next := pickEffect(current.Effects)
		if visited[next] {
			return append(chain, TruncationMarker) // cycle: signal incompleteness
		}
		if len(chain) == MaxChainHops {
			return append(chain, TruncationMarker) // depth bound reached
		}
		chain = append(chain, next)
		visited[next] = true

		nextEntry, ok := entries[next]
		if !ok || nextEntry.IsDirect {
			return chain
		}
		current = nextEntry
	}
	return chain
}

// pickEffect chooses the walk's next hop deterministically.
func pickEffect(effects []string) string {
	sorted := append([]string(nil), effects...)
	sort.Strings(sorted)
	return sorted[0]
}
