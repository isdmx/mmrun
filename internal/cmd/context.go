package cmd

import (
	"io"
	"os"

	"github.com/dmitriev/mmrun/internal/client"
	"github.com/dmitriev/mmrun/internal/config"
	"github.com/dmitriev/mmrun/internal/output"
	"github.com/dmitriev/mmrun/internal/session"
)

// appContext carries shared dependencies into command RunE functions.
type appContext struct {
	api         client.API
	outputMode  string
	defaultTeam string
	userID      string
}

// requireSession builds an authenticated appContext from the stored session.
func requireSession(outputMode string) (*appContext, error) {
	sess, err := session.Load()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &appContext{
		api:         client.NewWithToken(sess.ServerURL, sess.Token),
		outputMode:  outputMode,
		defaultTeam: cfg.DefaultTeam,
		userID:      sess.UserID,
	}, nil
}

// render writes a Result to stdout using the resolved output mode.
func (a *appContext) render(r output.Result) error {
	return output.New(a.outputMode, os.Stdout).Render(os.Stdout, r)
}

// stdoutFile returns os.Stdout when w is os.Stdout, else os.Stdout as a fallback
// for TTY detection when writing to a non-file writer (e.g. test buffers).
func stdoutFile(w io.Writer) *os.File {
	if f, ok := w.(*os.File); ok {
		return f
	}
	return os.Stdout
}
