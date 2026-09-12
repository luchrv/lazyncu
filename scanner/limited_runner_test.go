package scanner

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// gatedRunner counts concurrent Run calls and blocks each one on release.
type gatedRunner struct {
	release  chan struct{}
	inFlight atomic.Int32
	peak     atomic.Int32
}

func (g *gatedRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	n := g.inFlight.Add(1)
	for {
		p := g.peak.Load()
		if n <= p || g.peak.CompareAndSwap(p, n) {
			break
		}
	}
	<-g.release
	g.inFlight.Add(-1)
	return []byte("ok"), nil
}

func TestLimitedRunnerNeverExceedsMax(t *testing.T) {
	// Arrange
	inner := &gatedRunner{release: make(chan struct{})}
	limited := NewLimitedRunner(inner, 3)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { _, _ = limited.Run(context.Background(), "", "ncu") })
	}
	waitFor(t, func() bool { return inner.inFlight.Load() == 3 })

	// Act: release everything and let the waiters flow through
	close(inner.release)
	wg.Wait()

	// Assert
	if peak := inner.peak.Load(); peak != 3 {
		t.Errorf("peak concurrency = %d, want exactly 3", peak)
	}
}

func TestLimitedRunnerWaitersProceedAsSlotsFree(t *testing.T) {
	// Arrange: one slot, first call holds it
	inner := &gatedRunner{release: make(chan struct{})}
	limited := NewLimitedRunner(inner, 1)
	go func() { _, _ = limited.Run(context.Background(), "", "ncu") }()
	waitFor(t, func() bool { return inner.inFlight.Load() == 1 })

	done := make(chan struct{})
	go func() {
		_, _ = limited.Run(context.Background(), "", "ncu")
		close(done)
	}()

	// Assert: the second call waits while the slot is held
	select {
	case <-done:
		t.Fatal("second call ran while the only slot was held")
	case <-time.After(50 * time.Millisecond):
	}

	// Act
	close(inner.release)

	// Assert
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second call never proceeded after the slot freed")
	}
}

func TestLimitedRunnerCancelWhileWaiting(t *testing.T) {
	// Arrange: the only slot is held forever
	inner := &gatedRunner{release: make(chan struct{})}
	defer close(inner.release)
	limited := NewLimitedRunner(inner, 1)
	go func() { _, _ = limited.Run(context.Background(), "", "ncu") }()
	waitFor(t, func() bool { return inner.inFlight.Load() == 1 })
	ctx, cancel := context.WithCancel(context.Background())

	// Act
	errCh := make(chan error, 1)
	go func() {
		_, err := limited.Run(ctx, "", "ncu")
		errCh <- err
	}()
	cancel()

	// Assert
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled waiter never returned")
	}
	if inner.inFlight.Load() != 1 {
		t.Errorf("inner ran the cancelled command; in flight = %d", inner.inFlight.Load())
	}
}

func TestLimitedRunnerMaxBelowOneStillRuns(t *testing.T) {
	// Arrange
	inner := &gatedRunner{release: make(chan struct{})}
	close(inner.release)

	for _, max := range []int{0, -3} {
		// Act
		out, err := NewLimitedRunner(inner, max).Run(context.Background(), "", "ncu")

		// Assert
		if err != nil || string(out) != "ok" {
			t.Errorf("max=%d: Run() = %q, %v; want ok", max, out, err)
		}
	}
}

func TestLimitedRunnerWaitDoesNotEatTimeout(t *testing.T) {
	// Arrange: a 100 ms per-command timeout behind a single slot that is
	// held three times longer than the timeout.
	limited := NewLimitedRunner(ExecRunner{Timeout: 100 * time.Millisecond}, 1)
	limited.slots <- struct{}{}
	go func() {
		time.Sleep(300 * time.Millisecond)
		<-limited.slots
	}()

	// Act
	_, err := limited.Run(context.Background(), "", "true")

	// Assert: the command got its full timeout once it started
	if err != nil {
		t.Errorf("Run() error = %v, want the wait to not count against the timeout", err)
	}
}

// waitFor polls cond until it holds or the test deadline passes.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition never met")
		}
		time.Sleep(time.Millisecond)
	}
}
