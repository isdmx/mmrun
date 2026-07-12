package cmd

import (
	"context"
	"io"

	"github.com/dmitriev/mmrun/internal/output"
	"github.com/spf13/cobra"
)

func newMeCmd(outputMode *string) *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated account and status",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runMe(app, cmd.OutOrStdout())
		},
	}
}

func runMe(app *appContext, w io.Writer) error {
	ctx := context.Background()
	u, err := app.api.Me(ctx)
	if err != nil {
		return err
	}
	statusText := ""
	if s, err := app.api.Status(ctx, u.Id); err == nil && s != nil {
		statusText = s.Status
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
		},
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}
