package client

import (
	"errors"

	"github.com/mattermost/mattermost/server/public/model"
)

// StatusCode extracts the HTTP status code embedded in a Mattermost API error.
// It returns 0 when the error carries no status (e.g. a network-level error).
func StatusCode(err error) int {
	var appErr *model.AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}
	return 0
}
