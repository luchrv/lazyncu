// Package config persists user configuration (registered project paths and
// scan settings) as a TOML file, with immutable update operations.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// DefaultTimeoutMS is the scan timeout used when the config file sets none.
const DefaultTimeoutMS = 30000

const (
	// appDirName is the config directory name. The app was renamed from
	// ncu-tui; a leftover ~/.config/ncu-tui/ is deliberately not migrated.
	appDirName     = "lazyncu"
	configFileName = "config.toml"
	dirPerm        = 0o755
	filePerm       = 0o644
)

// Path is a user-registered filesystem location to scan.
type Path struct {
	Path string `toml:"path"`
}

// Config is the full persisted configuration. Update methods return new
// values and never mutate the receiver.
type Config struct {
	TimeoutMS int    `toml:"timeout_ms,omitempty"`
	Paths     []Path `toml:"paths,omitempty"`
}

// FilePath resolves the config file location: $XDG_CONFIG_HOME/lazyncu/config.toml,
// falling back to ~/.config/lazyncu/config.toml.
func FilePath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appDirName, configFileName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".config", appDirName, configFileName), nil
}

// Load reads the config file at path, creating an empty one (and its parent
// directory) on first launch. A malformed file is reported and left untouched.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg := Config{TimeoutMS: DefaultTimeoutMS}
		if saveErr := Save(path, cfg); saveErr != nil {
			return Config{}, fmt.Errorf("creating config file %s: %w", path, saveErr)
		}
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	if cfg.TimeoutMS == 0 {
		cfg.TimeoutMS = DefaultTimeoutMS
	}
	return cfg, nil
}

// Save writes cfg to path, creating parent directories as needed.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, data, filePerm); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}
	return nil
}

// AddPath returns a new Config with raw appended after tilde expansion and
// cleaning. It rejects paths that do not exist and duplicates.
func (c Config) AddPath(raw string) (Config, error) {
	expanded, err := expandTilde(raw)
	if err != nil {
		return Config{}, err
	}
	cleaned := filepath.Clean(expanded)

	if _, err := os.Stat(cleaned); err != nil {
		return Config{}, fmt.Errorf("path %s is not accessible: %w", cleaned, err)
	}
	for _, p := range c.Paths {
		if p.Path == cleaned {
			return Config{}, fmt.Errorf("path %s is already registered", cleaned)
		}
	}

	updated := c
	updated.Paths = append(slices.Clone(c.Paths), Path{Path: cleaned})
	return updated, nil
}

// RemovePath returns a new Config without the given path. Removing an
// unregistered path is a no-op.
func (c Config) RemovePath(path string) Config {
	updated := c
	updated.Paths = slices.DeleteFunc(slices.Clone(c.Paths), func(p Path) bool {
		return p.Path == path
	})
	return updated
}

func expandTilde(raw string) (string, error) {
	if raw != "~" && !strings.HasPrefix(raw, "~"+string(filepath.Separator)) {
		return raw, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expanding ~: %w", err)
	}
	return filepath.Join(home, raw[1:]), nil
}
