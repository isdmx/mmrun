package cmd

import (
	"errors"

	"github.com/dmitriev/mmrun/internal/client"
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
	switch client.StatusCode(err) {
	case 401, 403:
		return 2
	case 404:
		return 3
	}
	return 1
}
