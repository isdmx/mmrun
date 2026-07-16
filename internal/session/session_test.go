package session

import (
	"errors"
	"os"
	"testing"
)

func TestSaveLoad_RoundTripAndPerms(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", root)

	in := &Session{ServerURL: "https://mm.example.com", Token: "abc123", UserID: "u1"}
	if err := Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(pathForTest(t))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perm = %o, want 600", perm)
	}

	out, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.Token != in.Token || out.ServerURL != in.ServerURL || out.UserID != in.UserID {
		t.Errorf("round trip mismatch: %+v", out)
	}
}

func TestSaveLoad_Username(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := Save(&Session{ServerURL: "https://x", Token: "t", UserID: "u1", Username: "alice"}); err != nil {
		t.Fatal(err)
	}
	out, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if out.Username != "alice" {
		t.Errorf("Username = %q, want alice", out.Username)
	}
}

func TestSaveLoad_ContextName(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if err := Save(&Session{ServerURL: "https://x", Token: "t", UserID: "u1", ContextName: "work"}); err != nil {
		t.Fatal(err)
	}
	out, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if out.ContextName != "work" {
		t.Errorf("ContextName = %q, want work", out.ContextName)
	}
}

func TestLoad_MissingReturnsErrNoSession(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := Load(); !errors.Is(err, ErrNoSession) {
		t.Errorf("err = %v, want ErrNoSession", err)
	}
}

func TestClear(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	_ = Save(&Session{Token: "x"})
	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if _, err := Load(); !errors.Is(err, ErrNoSession) {
		t.Errorf("after clear err = %v, want ErrNoSession", err)
	}
}
