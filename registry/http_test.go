package registry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fallbackSpy records fallback use.
type fallbackSpy struct {
	calls int32
	md    Metadata
	err   error
}

func (f *fallbackSpy) Fetch(_ context.Context, _, _ string) (Metadata, error) {
	atomic.AddInt32(&f.calls, 1)
	return f.md, f.err
}

func (f *fallbackSpy) PublishTimes(_ context.Context, _, _ string) (map[string]string, error) {
	atomic.AddInt32(&f.calls, 1)
	return nil, f.err
}

func newResolverFor(registryURL string) *Resolver {
	r := &fakeRunner{responses: map[string][]byte{
		"npm config list --json": []byte(`{"registry":"` + registryURL + `","userconfig":"/nonexistent","globalconfig":"/nonexistent","@acme:registry":"` + registryURL + `scoped/"}`),
	}}
	resolver := NewResolver(r)
	resolver.env = func() []string {
		return []string{"npm_config_" + "//" + registryURL[len("http://"):] + "scoped/:_authToken=tok"}
	}
	return resolver
}

func TestHTTPFetcherRequestsAbbreviatedPackument(t *testing.T) {
	// Arrange
	var gotAccept, gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept, gotPath, gotAuth = r.Header.Get("Accept"), r.URL.EscapedPath(), r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"dist-tags":{"latest":"2.0.0"},"versions":{"1.0.0":{"version":"1.0.0"},"2.0.0":{"version":"2.0.0","engines":{"node":">=18"}}}}`))
	}))
	defer srv.Close()
	fallback := &fallbackSpy{err: errors.New("unused")}
	f := NewHTTPFetcher(newResolverFor(srv.URL+"/"), fallback, 4, time.Second)

	// Act
	md, err := f.Fetch(context.Background(), "/p/app", "@acme/pkg")

	// Assert
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if gotAccept != AbbreviatedAccept || gotPath != "/scoped/@acme%2Fpkg" || gotAuth != "Bearer tok" {
		t.Errorf("request = accept %q path %q auth %q", gotAccept, gotPath, gotAuth)
	}
	if md.DistTags["latest"] != "2.0.0" || len(md.Versions) != 2 {
		t.Errorf("metadata = %+v", md)
	}
	if atomic.LoadInt32(&fallback.calls) != 0 {
		t.Error("fallback used on a successful request")
	}
}

func TestHTTPFetcherFallsBackOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"Not found"}`))
	}))
	defer srv.Close()
	fallback := &fallbackSpy{md: Metadata{DistTags: map[string]string{"latest": "9.9.9"}, Versions: versions(Version{Version: "9.9.9"})}}
	f := NewHTTPFetcher(newResolverFor(srv.URL+"/"), fallback, 4, time.Second)

	md, err := f.Fetch(context.Background(), "", "private-pkg")
	_, timesErr := f.PublishTimes(context.Background(), "", "private-pkg")

	if err != nil || md.DistTags["latest"] != "9.9.9" {
		t.Errorf("Fetch() = %+v, %v, want the fallback's metadata", md, err)
	}
	if timesErr != nil || atomic.LoadInt32(&fallback.calls) != 2 {
		t.Errorf("fallback calls = %d (times err %v), want both calls delegated", fallback.calls, timesErr)
	}
}

func TestHTTPFetcherPublishTimesUsesFullPackument(t *testing.T) {
	var gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte(`{"versions":{"1.0.0":{}},"time":{"1.0.0":"2020-01-02T00:00:00Z"}}`))
	}))
	defer srv.Close()
	f := NewHTTPFetcher(newResolverFor(srv.URL+"/"), &fallbackSpy{err: errors.New("unused")}, 4, time.Second)

	times, err := f.PublishTimes(context.Background(), "", "pkg")

	if err != nil || times["1.0.0"] != "2020-01-02T00:00:00Z" || gotAccept != "application/json" {
		t.Errorf("PublishTimes() = %v, %v (accept %q)", times, err, gotAccept)
	}
}

func TestHTTPFetcherBoundsConcurrency(t *testing.T) {
	var inflight, peak int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&inflight, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if n <= p || atomic.CompareAndSwapInt32(&peak, p, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&inflight, -1)
		_, _ = w.Write([]byte(`{"dist-tags":{},"versions":{"1.0.0":{}}}`))
	}))
	defer srv.Close()
	f := NewHTTPFetcher(newResolverFor(srv.URL+"/"), &fallbackSpy{err: errors.New("unused")}, 2, time.Second)

	done := make(chan struct{})
	for i := range 8 {
		go func() {
			_, _ = f.Fetch(context.Background(), "", "pkg"+string(rune('a'+i)))
			done <- struct{}{}
		}()
	}
	for range 8 {
		<-done
	}
	if peak > 2 {
		t.Errorf("peak in-flight requests = %d, want <= 2", peak)
	}
}
