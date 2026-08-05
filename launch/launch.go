// Package launch resolves and classifies the optional positional CLI path
// argument against the registered paths, entirely before the TUI starts.
// Comparisons run on symlink-resolved forms; persisted paths are never
// rewritten.
package launch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/detect"
)

// Kind is the classification of the launch target against registered paths.
type Kind int

const (
	// KindNew is a target unrelated to every registered path.
	KindNew Kind = iota
	// KindEqual is a target identical to a registered path.
	KindEqual
	// KindContained is a target strictly inside a registered path.
	KindContained
	// KindParent is a target strictly containing one or more registered paths.
	KindParent
)

// Intent is the classified launch target the UI acts on.
type Intent struct {
	Kind Kind
	// Source is the panel source to select at startup: the covering
	// registered path for KindEqual/KindContained, the target itself for
	// KindParent/KindNew.
	Source string
	// TargetDir is the resolved target directory.
	TargetDir string
	// CoveredChildren lists registered paths now covered by a KindParent
	// target, in registration order.
	CoveredChildren []string
}

// Resolve turns the raw CLI argument into an absolute cleaned path: a
// leading ~ expands to the home directory and relative forms resolve
// against the current working directory.
func Resolve(raw string) (string, error) {
	expanded, err := expandTilde(raw)
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("resolving path %s: %w", raw, err)
	}
	return abs, nil
}

// Comparable returns the form of a path used for equality and containment
// checks: symlink-resolved when possible, cleaned otherwise. It never
// touches what is persisted.
func Comparable(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

// Classify compares the resolved target against the registered paths.
// Priority: equal, contained, parent, new — a target both inside one path
// and above another counts as contained.
func Classify(registered []config.Path, target string) Intent {
	resolvedTarget := Comparable(target)

	var covered []string
	for _, reg := range registered {
		resolvedReg := Comparable(reg.Path)
		switch {
		case resolvedReg == resolvedTarget:
			return Intent{Kind: KindEqual, Source: reg.Path, TargetDir: target}
		case contains(resolvedReg, resolvedTarget):
			return Intent{Kind: KindContained, Source: reg.Path, TargetDir: target}
		case contains(resolvedTarget, resolvedReg):
			covered = append(covered, reg.Path)
		}
	}
	if len(covered) > 0 {
		return Intent{Kind: KindParent, Source: target, TargetDir: target, CoveredChildren: covered}
	}
	return Intent{Kind: KindNew, Source: target, TargetDir: target}
}

// Prepare validates the raw CLI argument and applies its config effect:
// KindParent and KindNew targets are registered and persisted; the other
// kinds leave the config untouched. The returned config is what the UI
// must run with.
func Prepare(cfg config.Config, cfgPath, raw string) (config.Config, Intent, error) {
	target, err := Resolve(raw)
	if err != nil {
		return config.Config{}, Intent{}, err
	}
	if err := validateTarget(target); err != nil {
		return config.Config{}, Intent{}, err
	}

	intent := Classify(cfg.Paths, target)
	if intent.Kind == KindEqual || intent.Kind == KindContained {
		return cfg, intent, nil
	}

	updated, err := cfg.AddPath(target)
	if err != nil {
		return config.Config{}, Intent{}, err
	}
	if err := config.Save(cfgPath, updated); err != nil {
		return config.Config{}, Intent{}, err
	}
	return updated, intent, nil
}

// validateTarget rejects targets that cannot become a scan source: missing
// paths, non-directories, and directories without any Node project in reach.
func validateTarget(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("path %s is not accessible: %w", target, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path %s is not a directory", target)
	}
	if !detect.HasNodeProject(target) {
		return fmt.Errorf("no Node project found at %s (no package.json at its root or nearby subdirectories)", target)
	}
	return nil
}

// contains reports whether child is strictly inside parent, comparing whole
// path segments so /projects never contains /projects-old.
func contains(parent, child string) bool {
	if parent == child {
		return false
	}
	prefix := parent
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(child, prefix)
}

func expandTilde(raw string) (string, error) {
	if raw != "~" && !strings.HasPrefix(raw, "~"+string(filepath.Separator)) {
		return raw, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expanding ~: %w", err)
	}
	return filepath.Join(home, raw[1:]), nil
}
