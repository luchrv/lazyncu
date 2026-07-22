package scanner

import (
	"context"
	"os/exec"
	"time"
)

// ExecRunner is the production Runner: it spawns real processes with a
// per-command timeout. Stdout is returned even on non-zero exit so callers
// can parse tools that emit valid JSON alongside failure codes.
type ExecRunner struct {
	Timeout time.Duration
}

// Run executes name with args in dir (empty dir = current directory).
func (r ExecRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if r.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Output()
}
