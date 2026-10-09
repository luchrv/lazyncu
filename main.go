// Command lazyncu is a read-only terminal dashboard for outdated npm
// dependencies.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/launch"
	"github.com/luchrv/lazyncu/memo"
	"github.com/luchrv/lazyncu/orchestrator"
	"github.com/luchrv/lazyncu/registry"
	"github.com/luchrv/lazyncu/scanner"
	"github.com/luchrv/lazyncu/ui"
	"github.com/luchrv/lazyncu/version"
)

// registryCacheFile holds registry metadata across launches, next to the
// config file.
const registryCacheFile = "registry-cache.json"

// auditCacheFile holds audit results keyed by lockfile fingerprint.
const auditCacheFile = "audit-cache.json"

func main() {
	// Version must print even with a broken config or missing ncu, so it
	// runs before config load and preflight.
	if wantsVersion(os.Args) {
		fmt.Println(version.Get())
		return
	}
	target, err := positionalPath(os.Args)
	if err == nil {
		err = run(target)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "lazyncu:", err)
		os.Exit(1)
	}
}

// wantsVersion reports whether the first CLI argument requests the version.
func wantsVersion(args []string) bool {
	return len(args) > 1 && (args[1] == "--version" || args[1] == "-version")
}

// positionalPath extracts the optional positional path argument: none is
// fine, one is the launch target, more is a usage error.
func positionalPath(args []string) (string, error) {
	rest := args[1:]
	switch len(rest) {
	case 0:
		return "", nil
	case 1:
		return rest[0], nil
	default:
		return "", fmt.Errorf("too many arguments\nusage: lazyncu [path]")
	}
}

func run(target string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfgPath, err := config.FilePath()
	if err != nil {
		return err
	}
	cfg, firstRun, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	var intent *ui.Launch
	if target != "" {
		cfg, intent, err = prepareLaunch(cfg, cfgPath, target)
		if err != nil {
			return err
		}
	}

	// One bound for every external command (npm config, npm view, npm ls,
	// npm audit) across all sources; the per-command timeout starts only once
	// a slot is held. Registry HTTP requests have their own bound.
	runner := scanner.NewLimitedRunner(
		scanner.ExecRunner{Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond},
		cfg.MaxParallel)
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	fetcher := registry.NewHTTPFetcher(registry.NewResolver(runner), registry.NewFetcher(runner),
		cfg.MaxRequests, timeout)
	cache := registry.NewCache(fetcher, registry.CacheOptions{
		Path: filepath.Join(filepath.Dir(cfgPath), registryCacheFile),
		TTL:  cfg.CacheTTLDuration(),
	})
	defer func() { _ = cache.Flush() }()
	sc := scanner.New(runner, cache)

	auditStore := memo.New[audit.Result](memo.Options{
		Path: filepath.Join(filepath.Dir(cfgPath), auditCacheFile),
		TTL:  cfg.CacheTTLDuration(),
	})
	defer func() { _ = auditStore.Flush() }()
	auditor := audit.Cached(auditStore, func(ctx context.Context, dir string, pm detect.PackageManager) audit.Result {
		return audit.Run(ctx, runner, dir, pm)
	})
	deps := orchestrator.Deps{Scanner: sc, Auditor: orchestrator.Auditor(auditor), Discoverer: orchestrator.DiscoverRepos}
	return ui.New(ctx, cfg, cfgPath, deps, firstRun, intent).Run()
}

// prepareLaunch classifies the positional path (registering it when new or
// a parent of registered paths) and maps the result to the UI's launch
// intent. Validation failures abort before the TUI opens.
func prepareLaunch(cfg config.Config, cfgPath, target string) (config.Config, *ui.Launch, error) {
	updated, intent, err := launch.Prepare(cfg, cfgPath, target)
	if err != nil {
		return config.Config{}, nil, err
	}
	l := &ui.Launch{Source: intent.Source}
	switch intent.Kind {
	case launch.KindContained:
		l.ProjectDir = intent.TargetDir
	case launch.KindParent:
		l.CoveredChildren = intent.CoveredChildren
	}
	return updated, l, nil
}
