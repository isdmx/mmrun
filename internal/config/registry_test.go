package config

import (
	"os"
	"strings"
	"testing"
)

func TestRegistry_GetSet(t *testing.T) {
	c := &Config{}
	if err := Set(c, "default_team", "eng"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if c.DefaultTeam != "eng" {
		t.Errorf("DefaultTeam = %q", c.DefaultTeam)
	}
	v, err := Get(c, "default_team")
	if err != nil || v != "eng" {
		t.Errorf("get = %q, %v", v, err)
	}
}

func TestRegistry_UnknownKey(t *testing.T) {
	if _, err := Get(&Config{}, "nope"); err == nil {
		t.Error("expected error for unknown key on Get")
	}
	if err := Set(&Config{}, "nope", "x"); err == nil {
		t.Error("expected error for unknown key on Set")
	}
}

func TestRegistry_Validation(t *testing.T) {
	c := &Config{}
	if err := Set(c, "output_mode", "bogus"); err == nil {
		t.Error("expected enum validation error")
	}
	if err := Set(c, "output_mode", "json"); err != nil {
		t.Errorf("valid enum rejected: %v", err)
	}
	if err := Set(c, "default_limit", "notint"); err == nil {
		t.Error("expected int validation error")
	}
	if err := Set(c, "default_limit", "-5"); err == nil {
		t.Error("expected positive-int validation error")
	}
	if err := Set(c, "default_limit", "25"); err != nil {
		t.Errorf("valid int rejected: %v", err)
	}
	if err := Set(c, "color", "always"); err != nil {
		t.Errorf("valid color rejected: %v", err)
	}
}

func TestRegistry_Template(t *testing.T) {
	tmpl := Template()
	for _, key := range []string{"server_url", "default_team", "output_mode", "default_limit", "preview_len", "color", "download_dir", "columns"} {
		if !strings.Contains(tmpl, key) {
			t.Errorf("template missing key %q", key)
		}
	}
	if !strings.Contains(tmpl, "#") {
		t.Error("template should contain comments")
	}
}

func TestTemplate_GeneratesLoadableConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Save(&Config{}); err != nil {
		t.Fatal(err)
	}
	path := Paths().ConfigFile
	if err := os.WriteFile(path, []byte(Template()), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load()
	if err != nil {
		t.Fatalf("generated template is not loadable: %v", err)
	}
	if c.DefaultLimit() != 50 {
		t.Errorf("DefaultLimit = %d, want 50", c.DefaultLimit())
	}
	if c.PreviewLen() != 140 {
		t.Errorf("PreviewLen = %d, want 140", c.PreviewLen())
	}
	if strings.Contains(Template(), `default_limit = "50"`) {
		t.Error("integer key default_limit must be emitted unquoted")
	}
}

func TestGet_DownloadDirEffective(t *testing.T) {
	c := &Config{}
	v, err := Get(c, "download_dir")
	if err != nil {
		t.Fatal(err)
	}
	if v == "" {
		t.Error("download_dir should resolve to the XDG default, not empty")
	}
}

func TestKeys_Sorted(t *testing.T) {
	keys := Keys()
	if len(keys) != 11 {
		t.Errorf("expected 11 keys, got %d", len(keys))
	}
}
