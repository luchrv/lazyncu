package scanner

import "context"

// LimitedRunner bounds how many commands run at the same time across every
// caller sharing it (ncu scans, npm ls, npm audit). A slot is acquired
// before delegating, so an inner per-command timeout only starts once the
// command actually runs — waiting for a slot never eats into it.
type LimitedRunner struct {
	inner Runner
	slots chan struct{}
}

// NewLimitedRunner wraps inner with at most max concurrent commands; values
// below 1 are treated as 1.
func NewLimitedRunner(inner Runner, max int) LimitedRunner {
	if max < 1 {
		max = 1
	}
	return LimitedRunner{inner: inner, slots: make(chan struct{}, max)}
}

// Run waits for a free slot, then executes the command through the inner
// Runner. A context that ends while waiting returns ctx.Err() without
// running anything.
func (l LimitedRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	select {
	case l.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-l.slots }()
	return l.inner.Run(ctx, dir, name, args...)
}
