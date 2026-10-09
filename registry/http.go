package registry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// DefaultMaxRequests bounds concurrent registry requests when the config
// sets none.
const DefaultMaxRequests = 32

// maxPackumentBytes caps a packument body; the largest public packuments
// (full @types/node) are ~11 MB.
const maxPackumentBytes = 64 << 20

// HTTPFetcher is the production Fetcher: it requests packuments directly
// from the registry npm would use, with npm's credentials, proxy and TLS
// settings, and falls back to the npm view transport whenever a request
// fails, so parity with npm never regresses below the fallback.
type HTTPFetcher struct {
	resolver *Resolver
	fallback Fetcher
	timeout  time.Duration
	slots    chan struct{}
	mu       sync.Mutex
	clients  map[clientKey]*http.Client
}

type clientKey struct {
	strictSSL  bool
	caFile     string
	proxy      string
	httpsProxy string
}

// NewHTTPFetcher builds an HTTPFetcher. maxRequests bounds in-flight
// requests (values below 1 are treated as 1); timeout bounds each request.
func NewHTTPFetcher(resolver *Resolver, fallback Fetcher, maxRequests int, timeout time.Duration) *HTTPFetcher {
	if maxRequests < 1 {
		maxRequests = 1
	}
	return &HTTPFetcher{
		resolver: resolver,
		fallback: fallback,
		timeout:  timeout,
		slots:    make(chan struct{}, maxRequests),
		clients:  map[clientKey]*http.Client{},
	}
}

// Fetch retrieves the abbreviated packument for pkg as seen from dir.
func (f *HTTPFetcher) Fetch(ctx context.Context, dir, pkg string) (Metadata, error) {
	body, err := f.get(ctx, dir, pkg, AbbreviatedAccept)
	if err == nil {
		var md Metadata
		if md, err = parsePackument(body); err == nil {
			return md, nil
		}
	}
	return f.fallback.Fetch(ctx, dir, pkg)
}

// MinReleaseAge returns npm's raw min-release-age setting in effect in dir.
func (f *HTTPFetcher) MinReleaseAge(ctx context.Context, dir string) (string, error) {
	cfg, err := f.resolver.Resolve(ctx, dir)
	if err != nil {
		return "", err
	}
	return cfg.MinReleaseAge, nil
}

// PublishTimes retrieves the publish times from the full packument.
func (f *HTTPFetcher) PublishTimes(ctx context.Context, dir, pkg string) (map[string]string, error) {
	body, err := f.get(ctx, dir, pkg, "application/json")
	if err == nil {
		var times map[string]string
		if times, err = parsePackumentTimes(body); err == nil {
			return times, nil
		}
	}
	return f.fallback.PublishTimes(ctx, dir, pkg)
}

// get performs one bounded GET of pkg's packument with the given Accept
// header, returning the body of a 2xx response.
func (f *HTTPFetcher) get(ctx context.Context, dir, pkg, accept string) ([]byte, error) {
	cfg, err := f.resolver.Resolve(ctx, dir)
	if err != nil {
		return nil, err
	}
	registry, auth := cfg.Endpoint(pkg)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registry+packumentPath(pkg), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	switch {
	case auth.Token != "":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case auth.Basic != "":
		req.Header.Set("Authorization", "Basic "+auth.Basic)
	}

	select {
	case f.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-f.slots }()

	client, err := f.clientFor(cfg)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("GET %s: %s", req.URL.Redacted(), resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxPackumentBytes))
}

// packumentPath encodes a package name the way the registry expects:
// scoped names keep their "@" and encode the slash.
func packumentPath(pkg string) string {
	return strings.Replace(pkg, "/", "%2F", 1)
}

// clientFor memoizes one http.Client per distinct TLS/proxy configuration.
func (f *HTTPFetcher) clientFor(cfg NpmConfig) (*http.Client, error) {
	key := clientKey{strictSSL: cfg.StrictSSL, caFile: cfg.CAFile, proxy: cfg.Proxy, httpsProxy: cfg.HTTPSProxy}
	f.mu.Lock()
	defer f.mu.Unlock()
	if client, ok := f.clients[key]; ok {
		return client, nil
	}
	transport, err := newTransport(cfg)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: transport, Timeout: f.timeout}
	f.clients[key] = client
	return client, nil
}

// newTransport applies npm's strict-ssl, cafile and proxy settings. A
// disabled strict-ssl mirrors npm's own behavior for that user, which is
// why it is honored instead of ignored.
func newTransport(cfg NpmConfig) (*http.Transport, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	tlsConfig := &tls.Config{InsecureSkipVerify: !cfg.StrictSSL} //nolint:gosec // mirrors the user's npm strict-ssl=false
	if cfg.CAFile != "" {
		pem, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("reading npm cafile: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, errors.New("npm cafile holds no certificates")
		}
		tlsConfig.RootCAs = pool
	}
	transport.TLSClientConfig = tlsConfig
	if proxy := proxyURL(cfg); proxy != nil {
		transport.Proxy = http.ProxyURL(proxy)
	}
	return transport, nil
}

// proxyURL picks npm's https-proxy (registries are https) or proxy; nil
// keeps the environment-based default.
func proxyURL(cfg NpmConfig) *url.URL {
	for _, raw := range []string{cfg.HTTPSProxy, cfg.Proxy} {
		if raw == "" {
			continue
		}
		if u, err := url.Parse(raw); err == nil {
			return u
		}
	}
	return nil
}
