package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/memo"
)

func writeLock(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFingerprintFollowsLockfileContent(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, "package.json", `{"name":"a"}`)
	writeLock(t, dir, "package-lock.json", `{"lockfileVersion":3}`)

	first, err := Fingerprint(dir, detect.Npm)
	if err != nil {
		t.Fatal(err)
	}
	writeLock(t, dir, "package-lock.json", `{"lockfileVersion":3,"packages":{}}`)
	second, _ := Fingerprint(dir, detect.Npm)
	writeLock(t, dir, "package.json", `{"name":"b"}`)
	third, _ := Fingerprint(dir, detect.Npm)

	if first == second {
		t.Error("fingerprint unchanged after the lockfile changed")
	}
	if second != third {
		t.Error("fingerprint changed with package.json while a lockfile exists")
	}
	if _, err := Fingerprint(t.TempDir(), detect.Npm); err == nil {
		t.Error("Fingerprint() of an empty directory = nil error, want failure")
	}
}

func TestCachedReusesResultWhileLockfileUnchanged(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeLock(t, dir, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	calls := 0
	run := func(context.Context, string, detect.PackageManager) Result {
		calls++
		return Result{Status: StatusOK, Counters: Counters{High: calls}}
	}
	auditor := Cached(memo.New[Result](memo.Options{}), run)

	// Act
	first := auditor(context.Background(), dir, detect.Pnpm)
	second := auditor(context.Background(), dir, detect.Pnpm)
	writeLock(t, dir, "pnpm-lock.yaml", "lockfileVersion: '9.0'\npackages: {}\n")
	third := auditor(context.Background(), dir, detect.Pnpm)
	refreshed := auditor(memo.WithRefresh(context.Background()), dir, detect.Pnpm)

	// Assert
	if calls != 3 || first.Counters.High != 1 || second.Counters.High != 1 || third.Counters.High != 2 || refreshed.Counters.High != 3 {
		t.Errorf("calls = %d, highs = %d %d %d %d; want reuse, re-audit on lockfile change, re-audit on refresh",
			calls, first.Counters.High, second.Counters.High, third.Counters.High, refreshed.Counters.High)
	}
}

func TestCachedNeverStoresFailuresOrNotAvailable(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, "package-lock.json", "{}")
	calls := 0
	run := func(_ context.Context, _ string, pm detect.PackageManager) Result {
		calls++
		if pm == detect.Yarn {
			return Result{Status: StatusNotAvailable}
		}
		return Result{Status: StatusFailed, Err: "boom"}
	}
	path := filepath.Join(t.TempDir(), "audit-cache.json")
	store := memo.New[Result](memo.Options{Path: path, TTL: time.Hour})
	auditor := Cached(store, run)

	failed := auditor(context.Background(), dir, detect.Npm)
	auditor(context.Background(), dir, detect.Npm)
	auditor(context.Background(), dir, detect.Yarn)
	if err := store.Flush(); err != nil {
		t.Fatal(err)
	}

	if failed.Status != StatusFailed || failed.Err != "boom" {
		t.Errorf("failed result = %+v, want the failure surfaced", failed)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3 (failure retried, yarn bypasses the store)", calls)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("cache file written although nothing succeeded")
	}
}
