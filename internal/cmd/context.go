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
	botIDs         []string
	aliases        map[string]string
	markdown       bool
}

// requireSession builds an authenticated appContext from the stored session and
// config preferences.
func requireSession(outputMode string) (*appContext, error) {
	d, err := envAuth()
	if err == nil {
		return initFromEnvAuth(d, outputMode)
	}
	return initFromSession(outputMode)
}

// initFromEnvAuth builds an appContext from environment variable credentials.
func initFromEnvAuth(d envAuthData, outputMode string) (*appContext, error) {
	cl := client.NewWithToken(d.url, d.token)
	u, uerr := cl.Me(context.Background())
	if uerr != nil {
		return nil, fmt.Errorf("env auth token validation failed: %w", uerr)
	}
	cfg, cfgerr := config.Load()
	var botIDs []string
	if bots, berr := cl.Bots(context.Background()); berr == nil {
		for _, b := range bots {
			botIDs = append(botIDs, b.UserId)
		}
	}
	prefs := configWithDefaults(cfg, cfgerr)
	var aliases map[string]string
	if cfg != nil && cfg.Contexts != nil && cfg.Contexts["default"].Aliases != nil {
		aliases = cfg.Contexts["default"].Aliases
	}
	return &appContext{
		api:            cl,
		outputMode:     outputMode,
		userID:         u.Id,
		username:       u.Username,
		mustLogin:      true,
		color:          "auto",
		theme:          prefs.theme,
		previewLen:     prefs.previewLen,
		defaultLimit:   prefs.defaultLimit,
		downloadDir:    prefs.downloadDir,
		columnsDefault: prefs.columnsDefault,
		format:         prefs.format,
		style:          prefs.style,
		timeFormat:     prefs.timeFormat,
		botIDs:         botIDs,
		aliases:        aliases,
		markdown:       prefs.markdown,
	}, nil
}

// initFromSession builds an appContext from the persistent session file.
func initFromSession(outputMode string) (*appContext, error) {
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

	var aliases map[string]string
	if cfg.Contexts != nil {
		ctxName := sess.ContextName
		if ctxName == "" {
			ctxName = "default"
		}
		if ctxCfg, ok := cfg.Contexts[ctxName]; ok {
			aliases = ctxCfg.Aliases
		}
	}

	var botIDs []string
	if bots, berr := cl.Bots(context.Background()); berr == nil {
		for _, b := range bots {
			botIDs = append(botIDs, b.UserId)
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
		botIDs:         botIDs,
		aliases:        aliases,
		markdown:       cfg.Markdown(),
	}, nil
}

type configPrefs struct {
	previewLen, defaultLimit         int
	downloadDir, columnsDefault      string
	format, theme, style, timeFormat string
	markdown                         bool
}

// configWithDefaults applies defaults to zero config values.
func configWithDefaults(cfg *config.Config, loadErr error) configPrefs {
	var p configPrefs
	if loadErr == nil && cfg != nil {
		p.previewLen = cfg.PreviewLen()
		p.defaultLimit = cfg.DefaultLimit()
		p.downloadDir = cfg.DownloadDir()
		p.columnsDefault = cfg.Columns
		p.format = cfg.Format()
		p.theme = cfg.Theme()
		p.style = cfg.Style()
		p.timeFormat = cfg.TimeFormat()
		p.markdown = cfg.Markdown()
	}
	if p.previewLen == 0 {
		p.previewLen = 140
	}
	if p.defaultLimit == 0 {
		p.defaultLimit = 50
	}
	if p.downloadDir == "" {
		p.downloadDir = config.Paths().DownloadDir
	}
	if p.format == "" {
		p.format = "table"
	}
	if p.style == "" {
		p.style = "table"
	}
	if p.timeFormat == "" {
		p.timeFormat = "rfc3339"
	}
	return p
}

// render writes a Result using the app's output mode, color, and highlight terms.
func (a *appContext) render(w io.Writer, res output.Result) error {
	return a.renderOpts(w, res, "", "", "", a.markdown)
}

// renderOpts renders with optional per-command format, style, time-format overrides,
// and a markdown flag.
func (a *appContext) renderOpts(w io.Writer, res output.Result, format, style, timeFormat string, markdown bool) error {
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
	if timeFormat != "" {
		opts.TimeFormat = timeFormat
	}
	opts.Markdown = markdown
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
	if expanded, ok := a.aliases[ref]; ok {
		ref = expanded
	}
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
