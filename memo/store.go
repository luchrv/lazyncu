// Package memo provides a keyed memoization store shared by the registry and
// audit caches: concurrent requests for one key share a single in-flight
// call, successful results are reused for the process lifetime (or until a
// refresh is requested through the context), and results can be persisted
// as JSON and reused across launches while younger than a TTL. Failures are
// never retained.
package memo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// saveDebounce batches persistence writes while results are landing.
const saveDebounce = 500 * time.Millisecond

type refreshKey struct{}

// WithRefresh marks a context as requesting fresh data: cached entries are
// bypassed and replaced by new calls. The UI uses it for explicit rescans.
func WithRefresh(ctx context.Context) context.Context {
	return context.WithValue(ctx, refreshKey{}, true)
}

// RefreshRequested reports whether ctx carries WithRefresh.
func RefreshRequested(ctx context.Context) bool {
	v, _ := ctx.Value(refreshKey{}).(bool)
	return v
}

// Options configures persistence; a zero value keeps the store in memory
// only.
type Options struct {
	// Path is the JSON file holding entries across launches ("" = none).
	Path string
	// TTL bounds how long a persisted entry is served; <= 0 disables
	// persistence.
	TTL time.Duration
}

// Store memoizes values of type T by string key.
type Store[T any] struct {
	ttl  time.Duration
	path string
	// Now is the clock used for TTL checks; tests override it.
	Now func() time.Time

	mu        sync.Mutex
	entries   map[string]*entry[T]
	dirty     bool
	saveTimer *time.Timer
}

// entry is one call in progress or completed; done closes when val/err are
// set. persisted marks an entry loaded from disk, subject to the TTL.
type entry[T any] struct {
	done      chan struct{}
	val       T
	err       error
	fetchedAt time.Time
	persisted bool
}

func (e *entry[T]) completed() bool {
	select {
	case <-e.done:
		return true
	default:
		return false
	}
}

// New builds a Store, loading persisted entries when opts names a readable
// file. An unreadable or malformed file is ignored and overwritten by the
// next save; entries older than the TTL are dropped on load.
func New[T any](opts Options) *Store[T] {
	s := &Store[T]{ttl: opts.TTL, Now: time.Now, entries: map[string]*entry[T]{}}
	if opts.TTL > 0 && opts.Path != "" {
		s.path = opts.Path
		s.load()
	}
	return s
}

// Do returns the memoized value for key, calling fn when the key is absent,
// stale, failed before, or explicitly refreshed through ctx.
func (s *Store[T]) Do(ctx context.Context, key string, fn func() (T, error)) (T, error) {
	s.mu.Lock()
	if e, ok := s.entries[key]; ok && (!e.completed() || s.usable(ctx, e)) {
		s.mu.Unlock()
		<-e.done
		return e.val, e.err
	}
	e := &entry[T]{done: make(chan struct{})}
	s.entries[key] = e
	s.mu.Unlock()

	e.val, e.err = fn()
	e.fetchedAt = s.Now()
	s.mu.Lock()
	if e.err != nil {
		if s.entries[key] == e {
			delete(s.entries, key)
		}
	} else {
		s.markDirty()
	}
	s.mu.Unlock()
	close(e.done)
	return e.val, e.err
}

// usable reports whether a completed entry may be served for ctx: never
// under an explicit refresh, and a persisted entry only within the TTL.
func (s *Store[T]) usable(ctx context.Context, e *entry[T]) bool {
	if RefreshRequested(ctx) {
		return false
	}
	return !e.persisted || s.Now().Sub(e.fetchedAt) < s.ttl
}

// --- persistence ---

type persisted[T any] struct {
	FetchedAt time.Time `json:"fetched_at"`
	Value     T         `json:"value"`
}

func (s *Store[T]) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file map[string]persisted[T]
	if json.Unmarshal(data, &file) != nil {
		return
	}
	now := s.Now()
	for key, p := range file {
		if now.Sub(p.FetchedAt) >= s.ttl {
			continue
		}
		e := &entry[T]{done: make(chan struct{}), val: p.Value, fetchedAt: p.FetchedAt, persisted: true}
		close(e.done)
		s.entries[key] = e
	}
}

// markDirty schedules a debounced save; callers hold s.mu.
func (s *Store[T]) markDirty() {
	if s.path == "" {
		return
	}
	s.dirty = true
	if s.saveTimer == nil {
		s.saveTimer = time.AfterFunc(saveDebounce, func() { _ = s.Flush() })
	} else {
		s.saveTimer.Reset(saveDebounce)
	}
}

// Flush writes the successful entries to disk when anything changed since
// the last save. It is safe to call concurrently and at exit.
func (s *Store[T]) Flush() error {
	s.mu.Lock()
	if s.path == "" || !s.dirty {
		s.mu.Unlock()
		return nil
	}
	s.dirty = false
	file := make(map[string]persisted[T], len(s.entries))
	for key, e := range s.entries {
		if e.completed() && e.err == nil {
			file[key] = persisted[T]{FetchedAt: e.fetchedAt, Value: e.val}
		}
	}
	s.mu.Unlock()

	data, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("encoding cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return errors.Join(fmt.Errorf("replacing cache: %w", err), os.Remove(tmp))
	}
	return nil
}
