package registry

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luchrv/lazyncu/memo"
)

// countingFetcher records calls and replies with canned results.
type countingFetcher struct {
	fetches int32
	times   int32
	err     error
	latest  string
}

func (c *countingFetcher) Fetch(_ context.Context, _, _ string) (Metadata, error) {
	atomic.AddInt32(&c.fetches, 1)
	if c.err != nil {
		return Metadata{}, c.err
	}
	latest := c.latest
	if latest == "" {
		latest = "1.0.0"
	}
	return Metadata{DistTags: map[string]string{"latest": latest}, Versions: versions(Version{Version: latest})}, nil
}

func (c *countingFetcher) PublishTimes(_ context.Context, _, _ string) (map[string]string, error) {
	atomic.AddInt32(&c.times, 1)
	return map[string]string{"1.0.0": "2020-01-02T00:00:00Z"}, nil
}

func TestCacheFetchesEachPackageOnce(t *testing.T) {
	inner := &countingFetcher{}
	cache := NewCache(inner, CacheOptions{})
	const callers = 20

	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := cache.Fetch(context.Background(), "", "lodash"); err != nil {
				t.Error(err)
			}
			if _, err := cache.PublishTimes(context.Background(), "", "lodash"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if _, err := cache.Fetch(context.Background(), "/other/dir", "chalk"); err != nil {
		t.Fatal(err)
	}

	if got := atomic.LoadInt32(&inner.fetches); got != 2 {
		t.Errorf("Fetch calls = %d, want 2 (one per distinct package)", got)
	}
	if got := atomic.LoadInt32(&inner.times); got != 1 {
		t.Errorf("PublishTimes calls = %d, want 1", got)
	}
}

func TestCacheDoesNotRetainFailures(t *testing.T) {
	inner := &countingFetcher{err: errors.New("network down")}
	cache := NewCache(inner, CacheOptions{})

	if _, err := cache.Fetch(context.Background(), "", "lodash"); err == nil {
		t.Fatal("Fetch() = nil, want the inner error")
	}
	inner.err = nil
	if _, err := cache.Fetch(context.Background(), "", "lodash"); err != nil {
		t.Fatalf("Fetch() after recovery error = %v", err)
	}
	if got := atomic.LoadInt32(&inner.fetches); got != 2 {
		t.Errorf("Fetch calls = %d, want 2 (failure retried)", got)
	}
}

func TestCacheRefreshBypassesEntry(t *testing.T) {
	inner := &countingFetcher{latest: "1.0.0"}
	cache := NewCache(inner, CacheOptions{})
	if _, err := cache.Fetch(context.Background(), "", "lodash"); err != nil {
		t.Fatal(err)
	}
	inner.latest = "2.0.0"

	refreshed, err := cache.Fetch(memo.WithRefresh(context.Background()), "", "lodash")
	if err != nil {
		t.Fatal(err)
	}
	again, _ := cache.Fetch(context.Background(), "", "lodash")

	if refreshed.DistTags["latest"] != "2.0.0" || again.DistTags["latest"] != "2.0.0" {
		t.Errorf("latest after refresh = %q then %q, want 2.0.0 fetched once and reused", refreshed.DistTags["latest"], again.DistTags["latest"])
	}
	if got := atomic.LoadInt32(&inner.fetches); got != 2 {
		t.Errorf("Fetch calls = %d, want 2", got)
	}
}

func TestCachePersistsEntriesWithinTTL(t *testing.T) {
	// Arrange: a first process fills and flushes the cache
	path := filepath.Join(t.TempDir(), "registry-cache.json")
	first := NewCache(&countingFetcher{latest: "1.0.0"}, CacheOptions{Path: path, TTL: time.Hour})
	if _, err := first.Fetch(context.Background(), "", "lodash"); err != nil {
		t.Fatal(err)
	}
	if _, err := first.PublishTimes(context.Background(), "", "lodash"); err != nil {
		t.Fatal(err)
	}
	if err := first.Flush(); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file not written: %v", err)
	}

	// Act: a second process within the TTL
	inner := &countingFetcher{latest: "2.0.0"}
	second := NewCache(inner, CacheOptions{Path: path, TTL: time.Hour})
	md, err := second.Fetch(context.Background(), "", "lodash")
	if err != nil {
		t.Fatal(err)
	}
	times, _ := second.PublishTimes(context.Background(), "", "lodash")

	// Assert: served from disk, no fetch
	if md.DistTags["latest"] != "1.0.0" || times["1.0.0"] == "" {
		t.Errorf("persisted entry = %+v / %v, want the first process's data", md, times)
	}
	if got := atomic.LoadInt32(&inner.fetches) + atomic.LoadInt32(&inner.times); got != 0 {
		t.Errorf("inner calls = %d, want 0 within the TTL", got)
	}
}

func TestCacheExpiredEntriesAreRefetched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-cache.json")
	first := NewCache(&countingFetcher{latest: "1.0.0"}, CacheOptions{Path: path, TTL: time.Hour})
	if _, err := first.Fetch(context.Background(), "", "lodash"); err != nil {
		t.Fatal(err)
	}
	if err := first.Flush(); err != nil {
		t.Fatal(err)
	}

	inner := &countingFetcher{latest: "2.0.0"}
	second := NewCache(inner, CacheOptions{Path: path, TTL: time.Hour})
	second.meta.Now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	md, err := second.Fetch(context.Background(), "", "lodash")

	if err != nil || md.DistTags["latest"] != "2.0.0" || atomic.LoadInt32(&inner.fetches) != 1 {
		t.Errorf("expired entry: md=%+v err=%v fetches=%d, want a fresh fetch", md, err, inner.fetches)
	}
}

func TestCacheZeroTTLKeepsMemoryOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-cache.json")
	cache := NewCache(&countingFetcher{}, CacheOptions{Path: path, TTL: 0})
	if _, err := cache.Fetch(context.Background(), "", "lodash"); err != nil {
		t.Fatal(err)
	}
	if err := cache.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("cache file written with cache disabled")
	}
}

func TestCacheIgnoresMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registry-cache.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	inner := &countingFetcher{}
	cache := NewCache(inner, CacheOptions{Path: path, TTL: time.Hour})
	if _, err := cache.Fetch(context.Background(), "", "lodash"); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&inner.fetches) != 1 {
		t.Errorf("fetches = %d, want 1 after ignoring the malformed file", inner.fetches)
	}
}
