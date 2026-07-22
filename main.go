// Command ncu-tui is a read-only terminal dashboard for npm-check-updates.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/luchrv/ncu-tui/audit"
	"github.com/luchrv/ncu-tui/config"
	"github.com/luchrv/ncu-tui/detect"
	"github.com/luchrv/ncu-tui/scanner"
	"github.com/luchrv/ncu-tui/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ncu-tui:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfgPath, err := config.FilePath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	runner := scanner.ExecRunner{Timeout: time.Duration(cfg.TimeoutMS) * time.Millisecond}
	sc := scanner.New(runner)

	if err := sc.Preflight(ctx); err != nil {
		return err
	}

	auditor := func(ctx context.Context, dir string, pm detect.PackageManager) audit.Result {
		return audit.Run(ctx, runner, dir, pm)
	}
	return ui.New(ctx, cfg, cfgPath, sc, auditor).Run()
}
