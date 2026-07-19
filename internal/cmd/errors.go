package cmd

import (
	"errors"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/isdmx/mmrun/internal/session"
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

// friendlyMsg returns a user-facing suggestion for known HTTP status codes.
func friendlyMsg(err error) string {
	code := client.StatusCode(err)
	switch code {
	case 401:
		return "Authentication failed. Re-authenticate with 'mmrun auth login'."
	case 403:
		return "Forbidden. You may not have access to this resource."
	case 404:
		return "Not found. Try searching for the resource or check the ID."
	default:
		return ""
	}
}
