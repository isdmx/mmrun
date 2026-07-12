package config

import (
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
