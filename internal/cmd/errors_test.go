package cmd

import (
	"errors"
	"testing"

	"github.com/dmitriev/mmrun/internal/session"
)

func TestExitCode(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Errorf("nil err = %d, want 0", got)
	}
	if got := ExitCode(session.ErrNoSession); got != 2 {
		t.Errorf("no-session err = %d, want 2", got)
	}
	if got := ExitCode(errors.New("boom")); got != 1 {
		t.Errorf("generic err = %d, want 1", got)
	}
}
