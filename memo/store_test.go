package memo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoSharesInFlightCallAndReusesResult(t *testing.T) {
	store := New[string](Options{})
	var calls int32
	fn := func() (string, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(10 * time.Millisecond)
		return "v", nil
	}

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got, err := store.Do(context.Background(), "k", fn); err != nil || got != "v" {
				t.Errorf("Do() = %q, %v", got, err)
			}
		}()
	}
	wg.Wait()
	if _, err := store.Do(context.Background(), "k", fn); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("fn called %d times, want 1", calls)
	}
}

func TestDoDoesNotRetainFailures(t *testing.T) {
	store := New[string](Options{})
	fail := errors.New("down")
	if _, err := store.Do(context.Background(), "k", func() (string, error) { return "", fail }); !errors.Is(err, fail) {
		t.Fatalf("Do() error = %v, want %v", err, fail)
	}
	got, err := store.Do(context.Background(), "k", func() (string, error) { return "ok", nil })
	if err != nil || got != "ok" {
		t.Errorf("Do() after failure = %q, %v, want a retry", got, err)
	}
}

func TestDoRefreshBypassesEntry(t *testing.T) {
	store := New[int](Options{})
	n := 0
	next := func() (int, error) { n++; return n, nil }
	if v, _ := store.Do(context.Background(), "k", next); v != 1 {
		t.Fatalf("first = %d", v)
	}
	if v, _ := store.Do(WithRefresh(context.Background()), "k", next); v != 2 {
		t.Errorf("refreshed = %d, want 2", v)
	}
	if v, _ := store.Do(context.Background(), "k", next); v != 2 {
		t.Errorf("after refresh = %d, want the refreshed value reused", v)
	}
}

func TestPersistenceWithinAndBeyondTTL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	first := New[string](Options{Path: path, TTL: time.Hour})
	if _, err := first.Do(context.Background(), "k", func() (string, error) { return "old", nil }); err != nil {
		t.Fatal(err)
	}
	if err := first.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}

	second := New[string](Options{Path: path, TTL: time.Hour})
	got, _ := second.Do(context.Background(), "k", func() (string, error) { return "new", nil })
	if got != "old" {
		t.Errorf("within TTL = %q, want the persisted value", got)
	}

	third := New[string](Options{Path: path, TTL: time.Hour})
	third.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	got, _ = third.Do(context.Background(), "k", func() (string, error) { return "new", nil })
	if got != "new" {
		t.Errorf("beyond TTL = %q, want a fresh call", got)
	}
}

func TestZeroTTLAndMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	off := New[string](Options{Path: path, TTL: 0})
	if _, err := off.Do(context.Background(), "k", func() (string, error) { return "v", nil }); err != nil {
		t.Fatal(err)
	}
	if err := off.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("file written with persistence disabled")
	}

	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := New[string](Options{Path: path, TTL: time.Hour})
	if got, err := store.Do(context.Background(), "k", func() (string, error) { return "v", nil }); err != nil || got != "v" {
		t.Errorf("Do() with malformed file = %q, %v", got, err)
	}
}
