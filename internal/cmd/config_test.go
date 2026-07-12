package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/config"
)

func TestConfigSetGet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := runConfigSet("default_team", "eng"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := runConfigGet("default_team")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "eng" {
		t.Errorf("get = %q, want eng", got)
	}
	c, _ := config.Load()
	if c.DefaultTeam != "eng" {
		t.Errorf("not persisted: %+v", c)
	}
}

func TestConfigSet_Invalid(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := runConfigSet("output_mode", "bogus"); err == nil {
		t.Error("expected validation error")
	}
}

func TestConfigGenerate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := runConfigGenerate(false); err != nil {
		t.Fatalf("generate: %v", err)
	}
	path := filepath.Join(dir, "mmrun", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "default_team") {
		t.Error("template missing keys")
	}
	if err := runConfigGenerate(false); err == nil {
		t.Error("expected refusal when file exists")
	}
	if err := runConfigGenerate(true); err != nil {
		t.Errorf("force failed: %v", err)
	}
}
