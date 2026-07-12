package config

import (
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

func TestKeys_Sorted(t *testing.T) {
	keys := Keys()
	if len(keys) != 8 {
		t.Errorf("expected 8 keys, got %d", len(keys))
	}
}
