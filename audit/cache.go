package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"github.com/luchrv/lazyncu/detect"
	"github.com/luchrv/lazyncu/memo"
)

// Runner audits one project directory (the signature orchestrator.Auditor
// expects).
type Runner func(ctx context.Context, dir string, pm detect.PackageManager) Result

// lockfiles lists, per package manager, the files whose content determines
// the audit result, most authoritative first.
var lockfiles = map[detect.PackageManager][]string{
	detect.Npm:  {"package-lock.json", "npm-shrinkwrap.json", "package.json"},
	detect.Pnpm: {"pnpm-lock.yaml", "package.json"},
}

// Fingerprint identifies a project's auditable input: its directory plus the
// SHA-256 of its lockfile (or package.json when no lockfile exists), so a
// cached result is reused only while the lockfile is unchanged.
func Fingerprint(dir string, pm detect.PackageManager) (string, error) {
	for _, name := range lockfiles[pm] {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		return dir + "|" + name + "|" + hex.EncodeToString(sum[:]), nil
	}
	return "", errors.New("no lockfile or package.json to fingerprint")
}

// errFailed keeps a failed audit out of the store so the next request
// retries it.
var errFailed = errors.New("audit failed")

// Cached wraps run with a memo.Store keyed by Fingerprint: a project whose
// lockfile is unchanged is served from the store (and from disk across
// launches, within the TTL) instead of auditing again. Not-available and
// failed results are never stored.
func Cached(store *memo.Store[Result], run Runner) Runner {
	return func(ctx context.Context, dir string, pm detect.PackageManager) Result {
		key, err := Fingerprint(dir, pm)
		if err != nil || Deferred(pm).Status == StatusNotAvailable {
			return run(ctx, dir, pm)
		}
		res, err := store.Do(ctx, key, func() (Result, error) {
			res := run(ctx, dir, pm)
			if res.Status != StatusOK {
				return res, errFailed
			}
			return res, nil
		})
		if errors.Is(err, errFailed) {
			return res
		}
		return res
	}
}
