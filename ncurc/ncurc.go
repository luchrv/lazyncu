// Package ncurc reads a project's .ncurc configuration and exposes the subset
// of options lazyncu supports, warning when a project carries options outside
// that subset or an executable configuration it cannot run.
package ncurc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Options is the supported subset of a project's .ncurc configuration.
type Options struct {
	Pre           bool
	Deprecated    *bool
	Target        string
	Filter        []string
	Reject        []string
	FilterVersion []string
	RejectVersion []string
}

// Result is a loaded configuration plus any parity warnings.
type Result struct {
	Options  Options
	Warnings []string
}

// dataFiles are the supported (non-executable) config filenames, in lookup
// priority order. `.ncurc.yaml` and `.ncurc.yml` are YAML; `.ncurc` too.
var dataFiles = []string{".ncurc", ".ncurc.json", ".ncurc.yaml", ".ncurc.yml"}

// execFiles are executable configs npm-check-updates would run; lazyncu does
// not support them.
var execFiles = []string{".ncurc.js", ".ncurc.cjs", ".ncurc.mjs"}

// supported lists the option keys that affect lazyncu's scan.
var supported = map[string]bool{
	"pre":           true,
	"deprecated":    true,
	"target":        true,
	"filter":        true,
	"reject":        true,
	"filterVersion": true,
	"rejectVersion": true,
}

// Load finds the nearest supported config for dir (searching dir and its
// ancestors) and decodes it. A missing config yields a zero Result. It never
// returns an error: an unreadable file degrades to defaults with a warning.
func Load(dir string) Result {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	for current := abs; ; {
		if res, found := loadFrom(current); found {
			return res
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	// No reachable data config; report an executable one if present anywhere.
	for current := abs; ; {
		if name, ok := firstExisting(current, execFiles); ok {
			return Result{Warnings: []string{
				"executable .ncurc (" + name + ") is not supported; using default options",
			}}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return Result{}
}

// loadFrom decodes the first supported data file present in dir, reporting
// whether one was found.
func loadFrom(dir string) (Result, bool) {
	name, ok := firstExisting(dir, dataFiles)
	if !ok {
		return Result{}, false
	}
	result := Result{}
	if _, hasExec := firstExisting(dir, execFiles); hasExec {
		result.Warnings = append(result.Warnings,
			"executable .ncurc in "+dir+" is ignored; using the data config")
	}

	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		result.Warnings = append(result.Warnings, "could not read "+name+": "+err.Error())
		return result, true
	}
	values := map[string]any{}
	if strings.HasSuffix(name, ".json") {
		err = json.Unmarshal(raw, &values)
	} else {
		err = yaml.Unmarshal(raw, &values)
	}
	if err != nil {
		result.Warnings = append(result.Warnings, "could not parse "+name+": "+err.Error())
		return result, true
	}

	result.Options = decode(values)
	result.Warnings = append(result.Warnings, outOfScopeWarnings(values)...)
	return result, true
}

// decode maps the supported keys onto Options.
func decode(values map[string]any) Options {
	var opts Options
	if v, ok := values["pre"].(bool); ok {
		opts.Pre = v
	}
	if v, ok := values["deprecated"].(bool); ok {
		opts.Deprecated = &v
	}
	if v, ok := values["target"].(string); ok {
		opts.Target = v
	}
	opts.Filter = stringList(values["filter"])
	opts.Reject = stringList(values["reject"])
	opts.FilterVersion = stringList(values["filterVersion"])
	opts.RejectVersion = stringList(values["rejectVersion"])
	return opts
}

// outOfScopeWarnings reports keys npm-check-updates understands but lazyncu
// does not apply, so the user knows results may differ.
func outOfScopeWarnings(values map[string]any) []string {
	var warnings []string
	for key := range values {
		if supported[key] {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("unsupported .ncurc option %q ignored", key))
	}
	// Deterministic order for stable output.
	for i := 1; i < len(warnings); i++ {
		for j := i; j > 0 && warnings[j] < warnings[j-1]; j-- {
			warnings[j], warnings[j-1] = warnings[j-1], warnings[j]
		}
	}
	return warnings
}

// DistTag returns the target dist-tag when `target` names one ("@next" ->
// "next", true); levels like "minor" yield ("", false).
func (o Options) DistTag() (string, bool) {
	if strings.HasPrefix(o.Target, "@") && len(o.Target) > 1 {
		return o.Target[1:], true
	}
	return "", false
}

// AllowDeprecated returns the effective deprecated setting (default true).
func (o Options) AllowDeprecated() bool {
	return o.Deprecated == nil || *o.Deprecated
}

func firstExisting(dir string, names []string) (string, bool) {
	for _, name := range names {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
			return name, true
		}
	}
	return "", false
}

func stringList(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

// MatchesAny reports whether name matches any of the glob patterns (* and ?).
// An empty pattern list matches nothing.
func MatchesAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if globMatch(p, name) {
			return true
		}
	}
	return false
}

func globMatch(pattern, name string) bool {
	re, err := regexp.Compile("^" + globRegex(pattern) + "$")
	if err != nil {
		return pattern == name
	}
	return re.MatchString(name)
}

func globRegex(pattern string) string {
	var b strings.Builder
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	return b.String()
}
