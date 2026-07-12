package cmd

import (
	"errors"
	"strings"

	"github.com/dmitriev/mmrun/internal/session"
)

// ExitCode maps an error to a process exit code.
//
//	0 success, 1 general, 2 auth/session, 3 not-found.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, session.ErrNoSession) {
		return 2
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "unauthorized"), strings.Contains(msg, "session"), strings.Contains(msg, "token"):
		return 2
	case strings.Contains(msg, "not found"), strings.Contains(msg, "404"):
		return 3
	default:
		return 1
	}
}
