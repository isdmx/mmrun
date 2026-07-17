package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/isdmx/mmrun/internal/config"
	"github.com/isdmx/mmrun/internal/output"
	"github.com/isdmx/mmrun/internal/session"
)

// appContext carries shared dependencies into command RunE functions.
type appContext struct {
	api            client.API
	outputMode     string
	defaultTeam    string
	userID         string
	username       string
	mustLogin      bool
	color          string
	previewLen     int
	defaultLimit   int
	downloadDir    string
	columnsDefault string
	format         string
	theme          string
	style          string
	timeFormat     string
}

// requireSession builds an authenticated appContext from the stored session and
// config preferences.
func requireSession(outputMode string) (*appContext, error) {
	d, err := envAuth()
	if err == nil {
		cl := client.NewWithToken(d.url, d.token)
		u, uerr := cl.Me(context.Background())
		if uerr != nil {
			return nil, fmt.Errorf("env auth token validation failed: %w", uerr)
		}
		cfg, cfgerr := config.Load()
		var previewLen, defaultLimit int
		var downloadDir, columnsDefault, format, theme, style, timeFormat string
		if cfgerr == nil && cfg != nil {
			previewLen = cfg.PreviewLen()
			defaultLimit = cfg.DefaultLimit()
			downloadDir = cfg.DownloadDir()
			columnsDefault = cfg.Columns
			format = cfg.Format()
			theme = cfg.Theme()
			style = cfg.Style()
			timeFormat = cfg.TimeFormat()
		}
		if previewLen == 0 {
			previewLen = 140
		}
		if defaultLimit == 0 {
			defaultLimit = 50
		}
		if downloadDir == "" {
			downloadDir = config.Paths().DownloadDir
		}
		if format == "" {
			format = "table"
		}
		if style == "" {
			style = "table"
		}
		if timeFormat == "" {
			timeFormat = "rfc3339"
		}
		return &appContext{
			api:            cl,
			outputMode:     outputMode,
			userID:         u.Id,
			username:       u.Username,
			mustLogin:      true,
			color:          "auto",
			theme:          theme,
			previewLen:     previewLen,
			defaultLimit:   defaultLimit,
			downloadDir:    downloadDir,
			columnsDefault: columnsDefault,
			format:         format,
			style:          style,
			timeFormat:     timeFormat,
		}, nil
	}

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
	cl := client.NewWithToken(sess.ServerURL, sess.Token)

	username := sess.Username
	if username == "" {
		if u, uerr := cl.Me(context.Background()); uerr == nil && u != nil {
			username = u.Username
			sess.Username = username
			_ = session.Save(sess)
		}
	}

	return &appContext{
		api:            cl,
		outputMode:     outputMode,
		defaultTeam:    cfg.DefaultTeam,
		userID:         sess.UserID,
		username:       username,
		color:          cfg.Color(),
		theme:          cfg.Theme(),
		previewLen:     cfg.PreviewLen(),
		defaultLimit:   cfg.DefaultLimit(),
		downloadDir:    cfg.DownloadDir(),
		columnsDefault: cfg.Columns,
		format:         cfg.Format(),
		style:          cfg.Style(),
		timeFormat:     cfg.TimeFormat(),
	}, nil
}

// render writes a Result using the app's output mode, color, and highlight terms.
func (a *appContext) render(w io.Writer, res output.Result) error {
	return a.renderOpts(w, res, "", "")
}

func (a *appContext) renderWith(w io.Writer, res output.Result, format string) error {
	return a.renderOpts(w, res, format, "")
}

// renderOpts renders with optional per-command format and style overrides.
func (a *appContext) renderOpts(w io.Writer, res output.Result, format, style string) error {
	opts := output.Options{
		Color: a.color, Theme: a.theme,
		Format: a.format, Style: a.style, TimeFormat: a.timeFormat,
	}
	if format != "" {
		opts.Format = format
	}
	if style != "" {
		opts.Style = style
	}
	if a.username != "" {
		opts.Highlight = []string{"@" + a.username}
	}
	return output.NewWithOptions(a.outputMode, stdoutFile(w), opts).Render(w, res)
}

// stdoutFile returns os.Stdout when w is os.Stdout, else os.Stdout as a fallback
// for TTY detection when writing to a non-file writer (e.g. test buffers).
func stdoutFile(w io.Writer) *os.File {
	if f, ok := w.(*os.File); ok {
		return f
	}
	return os.Stdout
}

// resolveChannel resolves a channel reference, using teamOverride (from a
// command's --team flag) as the default team when set, otherwise the configured
// default team. A bare channel name still falls back to the user's sole team.
func (a *appContext) resolveChannel(ctx context.Context, ref, teamOverride string) (*model.Channel, error) {
	team := teamOverride
	if team == "" {
		team = a.defaultTeam
	}
	return a.api.ResolveChannel(ctx, ref, team, a.userID)
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

// reLogin prompts the user to re-authenticate when the session has expired.
// It reads y/N from stdin, runs the password login flow (with MFA fallback),
// and saves the new session token. Returns the new token, or an error when
// the user declines or login fails.
func reLogin() (string, error) {
	fmt.Fprintf(os.Stderr, "\nSession expired. Re-authenticate? (y/N): ")
	var answer string
	_, _ = fmt.Scanln(&answer)
	if answer != "y" && answer != "Y" && answer != "yes" {
		return "", fmt.Errorf("re-login declined")
	}
	sess, err := session.Load()
	if err != nil {
		return "", err
	}
	userID, err := promptLine("Username or email: ")
	if err != nil {
		return "", err
	}
	pass, err := promptSecret("Password: ")
	if err != nil {
		return "", err
	}
	c := client.New(sess.ServerURL)
	_, err = c.Login(context.Background(), userID, pass)
	if err != nil {
		if needsMFA(err.Error()) {
			mfa, merr := promptLine("MFA token: ")
			if merr != nil {
				return "", merr
			}
			_, err = c.LoginWithMFA(context.Background(), userID, pass, mfa)
		}
		if err != nil {
			return "", fmt.Errorf("re-login failed: %w", err)
		}
	}
	tok := c.Token()
	_ = session.Save(&session.Session{
		ServerURL: sess.ServerURL,
		Token:     tok,
		UserID:    sess.UserID,
		Username:  sess.Username,
	})
	usr, _ := c.Me(context.Background())
	if usr != nil {
		_ = session.Save(&session.Session{
			ServerURL: sess.ServerURL, Token: tok,
			UserID: usr.Id, Username: usr.Username,
		})
	}
	return tok, nil
}
