package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilePathUsesXDGConfigHome(t *testing.T) {
	// Arrange
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")

	// Act
	got, err := FilePath()

	// Assert
	if err != nil {
		t.Fatalf("FilePath() error = %v", err)
	}
	want := filepath.Join("/custom/xdg", "ncu-tui", "config.toml")
	if got != want {
		t.Errorf("FilePath() = %q, want %q", got, want)
	}
}

func TestFilePathFallsBackToHomeConfig(t *testing.T) {
	// Arrange
	t.Setenv("XDG_CONFIG_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	// Act
	got, err := FilePath()

	// Assert
	if err != nil {
		t.Fatalf("FilePath() error = %v", err)
	}
	want := filepath.Join(home, ".config", "ncu-tui", "config.toml")
	if got != want {
		t.Errorf("FilePath() = %q, want %q", got, want)
	}
}

func TestLoadCreatesFileOnFirstLaunch(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "nested", "config.toml")

	// Act
	cfg, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Paths) != 0 {
		t.Errorf("first launch Paths = %v, want empty", cfg.Paths)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("config file not created: %v", statErr)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "config.toml")
	original := Config{
		TimeoutMS: 60000,
		Paths:     []Path{{Path: "/projects/api"}, {Path: "/projects/web"}},
	}

	// Act
	if err := Save(path, original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.TimeoutMS != 60000 {
		t.Errorf("TimeoutMS = %d, want 60000", loaded.TimeoutMS)
	}
	if len(loaded.Paths) != 2 || loaded.Paths[0].Path != "/projects/api" || loaded.Paths[1].Path != "/projects/web" {
		t.Errorf("Paths = %+v, want the two saved paths in order", loaded.Paths)
	}
}

func TestLoadMalformedTOMLReturnsErrorWithPath(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[[paths]\nbroken"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)

	// Act
	_, err := Load(path)

	// Assert
	if err == nil {
		t.Fatal("Load() error = nil, want parse error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not mention config path %q", err, path)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Error("Load() modified a malformed config file; must leave it untouched")
	}
}

func TestTimeoutDefaultsTo30000(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	cfg, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.TimeoutMS != DefaultTimeoutMS {
		t.Errorf("TimeoutMS = %d, want default %d", cfg.TimeoutMS, DefaultTimeoutMS)
	}
}

func TestTimeoutOverride(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("timeout_ms = 60000\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Act
	cfg, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.TimeoutMS != 60000 {
		t.Errorf("TimeoutMS = %d, want 60000", cfg.TimeoutMS)
	}
}

func TestAddPathAppendsAndReturnsNewConfig(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	cfg := Config{TimeoutMS: DefaultTimeoutMS}

	// Act
	updated, err := cfg.AddPath(dir)

	// Assert
	if err != nil {
		t.Fatalf("AddPath() error = %v", err)
	}
	if len(updated.Paths) != 1 || updated.Paths[0].Path != dir {
		t.Errorf("updated.Paths = %+v, want [%s]", updated.Paths, dir)
	}
	if len(cfg.Paths) != 0 {
		t.Error("AddPath() mutated the original Config; must be immutable")
	}
}

func TestAddPathExpandsTilde(t *testing.T) {
	// Arrange
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{}

	// Act
	updated, err := cfg.AddPath("~")

	// Assert
	if err != nil {
		t.Fatalf("AddPath(~) error = %v", err)
	}
	if updated.Paths[0].Path != home {
		t.Errorf("AddPath(~) stored %q, want %q", updated.Paths[0].Path, home)
	}
}

func TestAddPathCleansPath(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	messy := dir + string(filepath.Separator) + "." + string(filepath.Separator)
	cfg := Config{}

	// Act
	updated, err := cfg.AddPath(messy)

	// Assert
	if err != nil {
		t.Fatalf("AddPath() error = %v", err)
	}
	if updated.Paths[0].Path != dir {
		t.Errorf("AddPath(%q) stored %q, want cleaned %q", messy, updated.Paths[0].Path, dir)
	}
}

func TestAddPathRejectsNonExistent(t *testing.T) {
	// Arrange
	cfg := Config{}

	// Act
	_, err := cfg.AddPath(filepath.Join(t.TempDir(), "does-not-exist"))

	// Assert
	if err == nil {
		t.Fatal("AddPath() error = nil, want rejection of non-existent path")
	}
}

func TestAddPathRejectsDuplicate(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	cfg := Config{}
	cfg, err := cfg.AddPath(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Act: same path via a messy but equivalent spelling
	_, err = cfg.AddPath(dir + string(filepath.Separator) + ".")

	// Assert
	if err == nil {
		t.Fatal("AddPath() error = nil, want duplicate rejection")
	}
}

func TestRemovePathReturnsNewConfig(t *testing.T) {
	// Arrange
	cfg := Config{Paths: []Path{{Path: "/a"}, {Path: "/b"}}}

	// Act
	updated := cfg.RemovePath("/a")

	// Assert
	if len(updated.Paths) != 1 || updated.Paths[0].Path != "/b" {
		t.Errorf("updated.Paths = %+v, want [/b]", updated.Paths)
	}
	if len(cfg.Paths) != 2 {
		t.Error("RemovePath() mutated the original Config; must be immutable")
	}
}

func TestRemovePathUnknownIsNoop(t *testing.T) {
	// Arrange
	cfg := Config{Paths: []Path{{Path: "/a"}}}

	// Act
	updated := cfg.RemovePath("/zzz")

	// Assert
	if len(updated.Paths) != 1 {
		t.Errorf("updated.Paths = %+v, want unchanged [/a]", updated.Paths)
	}
}

func TestSavePersistsImmediately(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = cfg.AddPath(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Act
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	reloaded, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(reloaded.Paths) != 1 || reloaded.Paths[0].Path != dir {
		t.Errorf("reloaded.Paths = %+v, want [%s]", reloaded.Paths, dir)
	}
}
