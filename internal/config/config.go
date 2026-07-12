// Package config resolves XDG-compliant file locations and loads/saves the
// TOML preferences file.
package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const appName = "mmrun"

// Config holds user preferences persisted to config.toml.
type Config struct {
	ServerURL   string `toml:"server_url"`
	DefaultTeam string `toml:"default_team"`
	OutputMode  string `toml:"output_mode"`
}

// PathSet holds resolved XDG file locations.
type PathSet struct {
	ConfigFile  string
	SessionFile string
	DownloadDir string
	CacheDir    string
}

func xdg(env, fallbackRel string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, fallbackRel)
}

// Paths resolves all XDG paths for the app.
func Paths() PathSet {
	cfg := xdg("XDG_CONFIG_HOME", ".config")
	state := xdg("XDG_STATE_HOME", filepath.Join(".local", "state"))
	cache := xdg("XDG_CACHE_HOME", ".cache")
	download := os.Getenv("XDG_DOWNLOAD_DIR")
	if download == "" {
		home, _ := os.UserHomeDir()
		download = filepath.Join(home, "Downloads")
	}
	return PathSet{
		ConfigFile:  filepath.Join(cfg, appName, "config.toml"),
		SessionFile: filepath.Join(state, appName, "session.json"),
		DownloadDir: download,
		CacheDir:    filepath.Join(cache, appName),
	}
}

// Load reads config.toml, returning a zero Config if the file is absent.
func Load() (*Config, error) {
	path := Paths().ConfigFile
	var c Config
	if _, err := toml.DecodeFile(path, &c); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}
	return &c, nil
}

// Save writes config.toml, creating parent directories as needed.
func Save(c *Config) error {
	path := Paths().ConfigFile
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(c); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
