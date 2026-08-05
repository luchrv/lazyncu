// Command lazyncu is a read-only terminal dashboard for npm-check-updates.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luchrv/lazyncu/audit"
	"github.com/luchrv/lazyncu/config"
	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/launch"
	"github.com/luchrv/lazyncu/scanner"
	"github.com/luchrv/lazyncu/ui"
	"github.com/luchrv/lazyncu/version"
)

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

	runner := scanner.ExecRunner{Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond}
	sc := scanner.New(runner)

	if err := sc.Preflight(ctx); err != nil {
		return err
	}

	auditor := func(ctx context.Context, dir string, pm detect.PackageManager) audit.Result {
		return audit.Run(ctx, runner, dir, pm)
	}
	return ui.New(ctx, cfg, cfgPath, sc, auditor, firstRun, intent).Run()
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
