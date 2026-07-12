package cmd

import (
	"context"
	"io"

	"github.com/dmitriev/mmrun/internal/output"
	"github.com/spf13/cobra"
)

func newTeamCmd(outputMode *string) *cobra.Command {
	team := &cobra.Command{Use: "team", Short: "Team operations"}
	team.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List teams you belong to",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runTeamList(app, cmd.OutOrStdout())
		},
	})
	return team
}

func runTeamList(app *appContext, w io.Writer) error {
	teams, err := app.api.TeamsForUser(context.Background(), app.userID)
	if err != nil {
		return err
	}
	res := output.Result{Title: "Teams", Columns: []string{"name", "display", "id"}}
	for _, t := range teams {
		res.Rows = append(res.Rows, output.Row{"name": t.Name, "display": t.DisplayName, "id": t.Id})
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}
