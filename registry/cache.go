package registry

import (
	"context"
	"time"

	"github.com/luchrv/lazyncu/memo"
)

// DefaultCacheTTL is how long persisted registry metadata is served without
// a new query when the config sets no cache_ttl.
const DefaultCacheTTL = time.Hour

// Cache memoizes a Fetcher per package through memo.Store: within one
// process an entry is reused until a refresh is requested (memo.WithRefresh);
// persisted entries are reused only while younger than the TTL. Entries are
// keyed by package name, so a name served by two different registries from
// different projects shares one entry.
type Cache struct {
	inner Fetcher
	// settings answers npm settings lookups (the HTTP fetcher's resolver);
	// nil reports no settings.
	settings Settings
	meta     *memo.Store[Metadata]
	times    *memo.Store[map[string]string]
}

// Settings exposes npm settings the scanner needs besides metadata.
type Settings interface {
	MinReleaseAge(ctx context.Context, dir string) (string, error)
}

// CacheOptions configures persistence; a zero value keeps the cache in
// memory only.
type CacheOptions struct {
	// Path is the JSON file holding metadata across launches ("" = none);
	// publish times go to a sibling file with a "-times" suffix.
	Path string
	// TTL bounds how long a persisted entry is served; <= 0 disables
	// persistence.
	TTL time.Duration
}

// NewCache wraps inner with per-package memoization, loading persisted
// entries when opts names a readable file.
func NewCache(inner Fetcher, opts CacheOptions) *Cache {
	c := &Cache{
		inner: inner,
		meta:  memo.New[Metadata](memo.Options{Path: opts.Path, TTL: opts.TTL}),
		times: memo.New[map[string]string](memo.Options{Path: timesPath(opts.Path), TTL: opts.TTL}),
	}
	if s, ok := inner.(Settings); ok {
		c.settings = s
	}
	return c
}

// timesPath derives the publish-times file from the metadata file.
func timesPath(path string) string {
	if path == "" {
		return ""
	}
	if ext := ".json"; len(path) > len(ext) && path[len(path)-len(ext):] == ext {
		return path[:len(path)-len(ext)] + "-times" + ext
	}
	return path + "-times"
}

// Fetch returns the cached metadata for pkg, fetching it when absent, stale
// or explicitly refreshed.
func (c *Cache) Fetch(ctx context.Context, dir, pkg string) (Metadata, error) {
	return c.meta.Do(ctx, pkg, func() (Metadata, error) { return c.inner.Fetch(ctx, dir, pkg) })
}

// PublishTimes returns the cached publish times for pkg, fetching them when
// absent, stale or explicitly refreshed.
func (c *Cache) PublishTimes(ctx context.Context, dir, pkg string) (map[string]string, error) {
	return c.times.Do(ctx, pkg, func() (map[string]string, error) { return c.inner.PublishTimes(ctx, dir, pkg) })
}

// MinReleaseAge delegates to the inner fetcher's settings; the resolver
// memoizes them, so nothing is cached here.
func (c *Cache) MinReleaseAge(ctx context.Context, dir string) (string, error) {
	if c.settings == nil {
		return "", nil
	}
	return c.settings.MinReleaseAge(ctx, dir)
}

// Flush persists both stores; safe to call at exit.
func (c *Cache) Flush() error {
	if err := c.meta.Flush(); err != nil {
		return err
	}
	return c.times.Flush()
}
