package registry

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// NpmConfig is the subset of npm configuration the HTTP client needs,
// resolved for one project directory with npm's own precedence (project,
// user, global and environment).
type NpmConfig struct {
	Registry        string
	ScopeRegistries map[string]string
	StrictSSL       bool
	CAFile          string
	Proxy           string
	HTTPSProxy      string
	// MinReleaseAge is npm's raw min-release-age setting ("" when unset).
	MinReleaseAge string
	// auth maps a nerf-darted registry prefix ("//host/path/") to its
	// credentials.
	auth map[string]Auth
}

// Auth is the credential npm would send to one registry prefix.
type Auth struct {
	Token string
	// Basic is the ready-to-use base64 user:password pair.
	Basic string
}

// Endpoint returns the registry URL (trailing slash) and credentials npm
// would use for pkg.
func (c NpmConfig) Endpoint(pkg string) (string, Auth) {
	reg := c.Registry
	if scope, _, ok := strings.Cut(pkg, "/"); ok && strings.HasPrefix(scope, "@") {
		if r, ok := c.ScopeRegistries[scope]; ok {
			reg = r
		}
	}
	if !strings.HasSuffix(reg, "/") {
		reg += "/"
	}
	return reg, c.authFor(reg)
}

// authFor walks the registry's nerf-dart prefixes from the most specific to
// the host alone, the way npm matches credentials.
func (c NpmConfig) authFor(registry string) Auth {
	u, err := url.Parse(registry)
	if err != nil || u.Host == "" {
		return Auth{}
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := len(segments); i >= 0; i-- {
		prefix := "//" + u.Host + "/"
		if i > 0 && segments[0] != "" {
			prefix += strings.Join(segments[:i], "/") + "/"
		}
		if a, ok := c.auth[prefix]; ok {
			return a
		}
	}
	return Auth{}
}

// Resolver resolves NpmConfig per project directory, memoized for the
// process lifetime. The global, user and environment layers come from one
// `npm config list --json` run (which applies npm's precedence among them);
// a project's own .npmrc is overlaid locally so no npm process runs per
// project. Credentials, which npm never prints, are read from the npmrc
// files npm reports, the project .npmrc and npm_config_* variables.
type Resolver struct {
	runner Runner
	env    func() []string
	base   resolved
	mu     sync.Mutex
	byDir  map[string]*resolved
}

type resolved struct {
	once sync.Once
	cfg  NpmConfig
	err  error
}

// baseConfig is the npm configuration without any project layer.
type baseConfig struct {
	cfg NpmConfig
	raw map[string]string
}

// NewResolver builds a Resolver that shells `npm config` through runner.
func NewResolver(runner Runner) *Resolver {
	return &Resolver{runner: runner, env: os.Environ, byDir: map[string]*resolved{}}
}

// Resolve returns the npm configuration effective in dir ("" = no project
// layer).
func (r *Resolver) Resolve(ctx context.Context, dir string) (NpmConfig, error) {
	r.mu.Lock()
	entry, ok := r.byDir[dir]
	if !ok {
		entry = &resolved{}
		r.byDir[dir] = entry
	}
	r.mu.Unlock()
	entry.once.Do(func() { entry.cfg, entry.err = r.resolve(ctx, dir) })
	return entry.cfg, entry.err
}

func (r *Resolver) resolve(ctx context.Context, dir string) (NpmConfig, error) {
	r.base.once.Do(func() { r.base.cfg, r.base.err = r.resolveBase(ctx) })
	if r.base.err != nil {
		return NpmConfig{}, r.base.err
	}
	raw := map[string]string{}
	if dir != "" {
		mergeNpmrc(raw, filepath.Join(dir, ".npmrc"))
	}
	return overlay(r.base.cfg, raw), nil
}

// resolveBase runs npm once for the global, user and environment layers.
func (r *Resolver) resolveBase(ctx context.Context) (NpmConfig, error) {
	out, err := r.runner.Run(ctx, "", "npm", "config", "list", "--json")
	if err != nil {
		return NpmConfig{}, fmt.Errorf("npm config list: %w", err)
	}
	listed, err := parseConfigList(out)
	if err != nil {
		return NpmConfig{}, err
	}
	cfg := NpmConfig{
		Registry:        listed.string("registry"),
		ScopeRegistries: map[string]string{},
		StrictSSL:       listed.bool("strict-ssl", true),
		CAFile:          listed.string("cafile"),
		Proxy:           listed.string("proxy"),
		HTTPSProxy:      listed.string("https-proxy"),
		MinReleaseAge:   listed.string("min-release-age"),
		auth:            map[string]Auth{},
	}
	if cfg.Registry == "" {
		return NpmConfig{}, errors.New("npm config reports no registry")
	}
	for key := range listed {
		if scope, ok := strings.CutSuffix(key, ":registry"); ok && strings.HasPrefix(scope, "@") {
			cfg.ScopeRegistries[scope] = listed.string(key)
		}
	}
	raw := map[string]string{}
	mergeNpmrc(raw, listed.string("globalconfig"))
	mergeNpmrc(raw, listed.string("userconfig"))
	mergeEnv(raw, r.env())
	cfg.auth = authEntries(raw)
	return cfg, nil
}

// overlay applies a project .npmrc's keys on top of base, the way npm's
// project layer wins over the user layer. Only the keys the client uses are
// interpreted; credentials are merged by prefix.
func overlay(base NpmConfig, raw map[string]string) NpmConfig {
	if len(raw) == 0 {
		return base
	}
	cfg := base
	cfg.ScopeRegistries = maps.Clone(base.ScopeRegistries)
	cfg.auth = maps.Clone(base.auth)
	for key, value := range raw {
		switch {
		case key == "registry":
			cfg.Registry = value
		case key == "strict-ssl":
			cfg.StrictSSL = value != "false"
		case key == "cafile":
			cfg.CAFile = value
		case key == "proxy":
			cfg.Proxy = value
		case key == "https-proxy":
			cfg.HTTPSProxy = value
		case key == "min-release-age":
			cfg.MinReleaseAge = value
		case strings.HasPrefix(key, "@") && strings.HasSuffix(key, ":registry"):
			cfg.ScopeRegistries[strings.TrimSuffix(key, ":registry")] = value
		}
	}
	for prefix, auth := range authEntries(raw) {
		cfg.auth[prefix] = auth
	}
	return cfg
}

// configList is the decoded `npm config list --json` document.
type configList map[string]json.RawMessage

func parseConfigList(out []byte) (configList, error) {
	start := bytes.IndexByte(out, '{')
	if start < 0 {
		return nil, errors.New("npm config list: no JSON in output")
	}
	var listed configList
	if err := json.Unmarshal(out[start:], &listed); err != nil {
		return nil, fmt.Errorf("npm config list: %w", err)
	}
	return listed, nil
}

func (l configList) string(key string) string { return stringField(l[key]) }

func (l configList) bool(key string, fallback bool) bool {
	var b bool
	if json.Unmarshal(l[key], &b) == nil {
		return b
	}
	return fallback
}

var envRef = regexp.MustCompile(`\$\{([^}]+)\}`)

// mergeNpmrc overlays the key/value pairs of one npmrc file, later files
// winning, expanding ${VAR} references the way npm does. A missing or
// unreadable file contributes nothing.
func mergeNpmrc(into map[string]string, file string) {
	if file == "" {
		return
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == ';' || line[0] == '#' {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		value = envRef.ReplaceAllStringFunc(value, func(ref string) string {
			return os.Getenv(ref[2 : len(ref)-1])
		})
		into[strings.TrimSpace(key)] = value
	}
}

// mergeEnv overlays npm_config_//… variables, which carry registry
// credentials in CI environments.
func mergeEnv(into map[string]string, environ []string) {
	for _, kv := range environ {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		lower := strings.ToLower(key)
		if rest, ok := strings.CutPrefix(lower, "npm_config_"); ok && strings.HasPrefix(rest, "//") {
			into[key[len("npm_config_"):]] = value
		}
	}
}

// authEntries groups the credential keys ("//host/:_authToken",
// "//host/:_auth", "//host/:username" + "//host/:_password") by prefix.
func authEntries(raw map[string]string) map[string]Auth {
	auth := map[string]Auth{}
	for key, value := range raw {
		if !strings.HasPrefix(key, "//") {
			continue
		}
		idx := strings.LastIndex(key, "/:")
		if idx < 0 {
			continue
		}
		prefix, field := key[:idx+1], key[idx+2:]
		entry := auth[prefix]
		switch field {
		case "_authToken":
			entry.Token = value
		case "_auth":
			entry.Basic = value
		case "username":
			entry.Basic = basicPair(value, decodedPassword(raw[prefix+":_password"]))
		case "_password":
			if user := raw[prefix+":username"]; user != "" {
				entry.Basic = basicPair(user, decodedPassword(value))
			}
		}
		auth[prefix] = entry
	}
	return auth
}

// decodedPassword decodes npm's base64-encoded _password, keeping the raw
// value when it is not valid base64.
func decodedPassword(encoded string) string {
	if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
		return string(decoded)
	}
	return encoded
}

func basicPair(user, password string) string {
	if user == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + password))
}
