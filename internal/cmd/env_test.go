package cmd

import (
	"os"
	"testing"
)

func TestEnvAuth_SetsSession(t *testing.T) {
	oldURL := os.Getenv("MMRUN_URL")
	oldTok := os.Getenv("MMRUN_TOKEN")
	t.Cleanup(func() {
		os.Setenv("MMRUN_URL", oldURL)
		os.Setenv("MMRUN_TOKEN", oldTok)
	})
	os.Setenv("MMRUN_URL", "https://mm.example.com")
	os.Setenv("MMRUN_TOKEN", "tok123")

	data, err := envAuth()
	if err != nil {
		t.Fatalf("envAuth: %v", err)
	}
	if data.url != "https://mm.example.com" || data.token != "tok123" {
		t.Errorf("env auth data = %+v", data)
	}
}

func TestEnvAuth_Empty(t *testing.T) {
	oldURL := os.Getenv("MMRUN_URL")
	oldTok := os.Getenv("MMRUN_TOKEN")
	t.Cleanup(func() {
		os.Setenv("MMRUN_URL", oldURL)
		os.Setenv("MMRUN_TOKEN", oldTok)
	})
	os.Unsetenv("MMRUN_URL")
	os.Unsetenv("MMRUN_TOKEN")
	if _, err := envAuth(); err == nil {
		t.Error("expected error when env vars are unset")
	}
}

func TestEnvAuth_MissingOne(t *testing.T) {
	oldURL := os.Getenv("MMRUN_URL")
	oldTok := os.Getenv("MMRUN_TOKEN")
	t.Cleanup(func() {
		os.Setenv("MMRUN_URL", oldURL)
		os.Setenv("MMRUN_TOKEN", oldTok)
	})
	os.Setenv("MMRUN_URL", "https://mm.example.com")
	os.Unsetenv("MMRUN_TOKEN")
	if _, err := envAuth(); err == nil {
		t.Error("expected error when only one env var is set")
	}
}
