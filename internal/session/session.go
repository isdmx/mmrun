// Package session persists the authenticated session (server, token, user) to
// an XDG state file with 0600 permissions.
package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/isdmx/mmrun/internal/config"
)

// ErrNoSession indicates no session file exists yet.
var ErrNoSession = errors.New("no active session; run 'mmrun auth login'")

// Session is the persisted, keyed session store. A single "default" context is
// used now; the map keeps the format stable for future multi-account support.
type Session struct {
	ServerURL string    `json:"server_url"`
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	SessionID string    `json:"session_id,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

type store struct {
	Current  string              `json:"current"`
	Contexts map[string]*Session `json:"contexts"`
}

func path() string { return config.Paths().SessionFile }

// Save writes the session as the "default" context with 0600 permissions.
func Save(s *Session) error {
	p := path()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	st := &store{Current: "default", Contexts: map[string]*Session{"default": s}}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// Load returns the current session, or ErrNoSession if none is stored.
func Load() (*Session, error) {
	data, err := os.ReadFile(path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoSession
		}
		return nil, err
	}
	var st store
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	s, ok := st.Contexts[st.Current]
	if !ok || s == nil || s.Token == "" {
		return nil, ErrNoSession
	}
	return s, nil
}

// Clear removes the session file.
func Clear() error {
	err := os.Remove(path())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
