package cmd

import (
	"context"
	"io"

	"github.com/isdmx/mmrun/internal/output"
	"github.com/spf13/cobra"
)

func newMeCmd(outputMode *string) *cobra.Command {
	var profile bool
	cmd := &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated account and status",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runMe(app, profile, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&profile, "profile", false, "include extended profile fields")
	return cmd
}

func runMe(app *appContext, profile bool, w io.Writer) error {
	ctx := context.Background()
	u, err := app.api.Me(ctx)
	if err != nil {
		return err
	}
	statusText := ""
	if s, err := app.api.Status(ctx, u.Id); err == nil && s != nil {
		statusText = s.Status
	}

	customStatus := ""
	if cs := u.GetCustomStatus(); cs != nil {
		customStatus = joinNonEmpty(cs.Emoji, cs.Text)
	}

	res := output.Result{
		Title:   "Account",
		Columns: []string{"field", "value"},
		Rows: []output.Row{
			{"field": "username", "value": u.Username},
			{"field": "nickname", "value": u.Nickname},
			{"field": "email", "value": u.Email},
			{"field": "name", "value": u.GetFullName()},
			{"field": "roles", "value": u.Roles},
			{"field": "status", "value": statusText},
			{"field": "custom_status", "value": customStatus},
			{"field": "timezone", "value": u.GetPreferredTimezone()},
		},
	}
	if profile {
		res.Rows = append(res.Rows,
			output.Row{"field": "position", "value": u.Position},
			output.Row{"field": "locale", "value": u.Locale},
			output.Row{"field": "id", "value": u.Id},
		)
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}

// joinNonEmpty joins the non-empty parts with a single space.
func joinNonEmpty(parts ...string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += " "
		}
		out += p
	}
	return out
}
