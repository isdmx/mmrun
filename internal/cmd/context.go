package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dmitriev/mmrun/internal/client"
	"github.com/dmitriev/mmrun/internal/config"
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
// When the requested output mode is "auto" (the default), a configured
// output_mode preference is applied.
func requireSession(outputMode string) (*appContext, error) {
	sess, err := session.Load()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if (outputMode == "" || outputMode == "auto") && cfg.OutputMode != "" {
		outputMode = cfg.OutputMode
	}
	return &appContext{
		api:         client.NewWithToken(sess.ServerURL, sess.Token),
		outputMode:  outputMode,
		defaultTeam: cfg.DefaultTeam,
		userID:      sess.UserID,
	}, nil
}

// stdoutFile returns os.Stdout when w is os.Stdout, else os.Stdout as a fallback
// for TTY detection when writing to a non-file writer (e.g. test buffers).
func stdoutFile(w io.Writer) *os.File {
	if f, ok := w.(*os.File); ok {
		return f
	}
	return os.Stdout
}

// resolveTeam maps a team name to its ID among the user's memberships. When
// name is empty it defaults to the sole team the user belongs to, and returns
// an error if the user is in multiple teams so the caller can prompt for one.
func (a *appContext) resolveTeam(ctx context.Context, name string) (id, resolvedName string, err error) {
	teams, err := a.api.TeamsForUser(ctx, a.userID)
	if err != nil {
		return "", "", err
	}
	if len(teams) == 0 {
		return "", "", fmt.Errorf("you are not a member of any team")
	}
	if name == "" {
		if len(teams) == 1 {
			return teams[0].Id, teams[0].Name, nil
		}
		names := make([]string, 0, len(teams))
		for _, t := range teams {
			names = append(names, t.Name)
		}
		return "", "", fmt.Errorf("multiple teams; specify --team (one of: %s)", strings.Join(names, ", "))
	}
	for _, t := range teams {
		if t.Name == name {
			return t.Id, t.Name, nil
		}
	}
	return "", "", fmt.Errorf("team %q not found among your memberships", name)
}
