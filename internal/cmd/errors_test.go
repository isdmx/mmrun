package cmd

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/isdmx/mmrun/internal/session"
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

func TestExitCode_HTTPStatus(t *testing.T) {
	unauth := &model.AppError{StatusCode: 401, Message: "unauthorized"}
	if got := ExitCode(unauth); got != 2 {
		t.Errorf("401 = %d, want 2", got)
	}
	forbidden := &model.AppError{StatusCode: 403, Message: "forbidden"}
	if got := ExitCode(forbidden); got != 2 {
		t.Errorf("403 = %d, want 2", got)
	}
	notFound := &model.AppError{StatusCode: 404, Message: "missing"}
	if got := ExitCode(notFound); got != 3 {
		t.Errorf("404 = %d, want 3", got)
	}
	server := &model.AppError{StatusCode: 500, Message: "boom"}
	if got := ExitCode(server); got != 1 {
		t.Errorf("500 = %d, want 1", got)
	}
}
