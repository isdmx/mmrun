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
	ServerURL     string `toml:"server_url"`
	DefaultTeam   string `toml:"default_team"`
	OutputMode    string `toml:"output_mode"`
	DefaultLimit_ int    `toml:"default_limit"` //nolint:revive // toml field paired with accessor method
	PreviewLen_   int    `toml:"preview_len"`   //nolint:revive // toml field paired with accessor method
	ColorMode     string `toml:"color"`
	DownloadDir_  string `toml:"download_dir"` //nolint:revive // toml field paired with accessor method
	Columns       string `toml:"columns"`
}

// DefaultLimit returns the configured page size, or 50.
func (c *Config) DefaultLimit() int {
	if c.DefaultLimit_ > 0 {
		return c.DefaultLimit_
	}
	return 50
}

// PreviewLen returns the configured message preview length, or 140.
func (c *Config) PreviewLen() int {
	if c.PreviewLen_ > 0 {
		return c.PreviewLen_
	}
	return 140
}

// Color returns the color mode: auto (default), always, or never.
func (c *Config) Color() string {
	if c.ColorMode == "" {
		return "auto"
	}
	return c.ColorMode
}

// DownloadDir returns the configured download directory, or the XDG default.
func (c *Config) DownloadDir() string {
	if c.DownloadDir_ != "" {
		return c.DownloadDir_
	}
	return Paths().DownloadDir
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
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := toml.NewEncoder(f).Encode(c); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
