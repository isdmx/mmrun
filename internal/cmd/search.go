package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dmitriev/mmrun/internal/output"
	"github.com/spf13/cobra"
)

func newSearchCmd(outputMode *string) *cobra.Command {
	var teamName string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search messages (server-side; supports Mattermost search modifiers)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			team := teamName
			if team == "" {
				team = app.defaultTeam
			}
			return runSearch(app, strings.Join(args, " "), team, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&teamName, "team", "", "team to search within (defaults to configured default team)")
	return cmd
}

func runSearch(app *appContext, query, teamName string, w io.Writer) error {
	ctx := context.Background()
	if teamName == "" {
		return fmt.Errorf("no team specified and no default team configured")
	}
	teams, err := app.api.TeamsForUser(ctx, app.userID)
	if err != nil {
		return err
	}
	var teamID string
	for _, t := range teams {
		if t.Name == teamName {
			teamID = t.Id
			break
		}
	}
	if teamID == "" {
		return fmt.Errorf("team %q not found among your memberships", teamName)
	}
	pl, err := app.api.Search(ctx, teamID, query, false)
	if err != nil {
		return err
	}
	res := output.Result{Title: "Search results", Columns: []string{"time", "user_id", "message"}}
	if pl != nil {
		for _, id := range pl.Order {
			p := pl.Posts[id]
			if p == nil {
				continue
			}
			res.Rows = append(res.Rows, output.Row{
				"time":    time.UnixMilli(p.CreateAt).Format(time.RFC3339),
				"user_id": p.UserId,
				"message": p.Message,
			})
		}
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}
