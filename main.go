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
	"github.com/luchrv/lazyncu/scanner"
	"github.com/luchrv/lazyncu/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyncu:", err)
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
