package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPaths_UsesXDGEnv(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "cfg"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))

	p := Paths()
	if want := filepath.Join(root, "cfg", "mmrun", "config.toml"); p.ConfigFile != want {
		t.Errorf("ConfigFile = %q, want %q", p.ConfigFile, want)
	}
	if want := filepath.Join(root, "state", "mmrun", "session.json"); p.SessionFile != want {
		t.Errorf("SessionFile = %q, want %q", p.SessionFile, want)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("XDG_STATE_HOME", root)

	in := &Config{ServerURL: "https://mm.example.com", DefaultTeam: "eng", OutputMode: "human"}
	if err := Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.ServerURL != in.ServerURL || out.DefaultTeam != in.DefaultTeam || out.OutputMode != in.OutputMode {
		t.Errorf("round trip mismatch: got %+v want %+v", out, in)
	}
}

func TestDefaults(t *testing.T) {
	c := &Config{}
	if c.DefaultLimit() != 50 {
		t.Errorf("DefaultLimit = %d, want 50", c.DefaultLimit())
	}
	if c.PreviewLen() != 140 {
		t.Errorf("PreviewLen = %d, want 140", c.PreviewLen())
	}
	if c.Color() != "auto" {
		t.Errorf("Color = %q, want auto", c.Color())
	}
	c2 := &Config{DefaultLimit_: 10, PreviewLen_: 80, ColorMode: "never"}
	if c2.DefaultLimit() != 10 || c2.PreviewLen() != 80 || c2.Color() != "never" {
		t.Errorf("overrides not honored: %+v", c2)
	}
}

func TestSave_Perms0600(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Save(&Config{DefaultTeam: "eng"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(Paths().ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config perm = %o, want 600", perm)
	}
}
