package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/dmitriev/mmrun/internal/output"
	"github.com/spf13/cobra"
)

func newChannelCmd(outputMode *string) *cobra.Command {
	var teamName string
	channel := &cobra.Command{Use: "channel", Short: "Channel operations"}
	list := &cobra.Command{
		Use:   "list",
		Short: "List channels in a team",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			team := teamName
			if team == "" {
				team = app.defaultTeam
			}
			return runChannelList(app, team, cmd.OutOrStdout())
		},
	}
	list.Flags().StringVar(&teamName, "team", "", "team name (defaults to configured default team)")
	channel.AddCommand(list)
	return channel
}

func runChannelList(app *appContext, teamName string, w io.Writer) error {
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
	channels, err := app.api.ChannelsForUser(ctx, teamID, app.userID)
	if err != nil {
		return err
	}
	res := output.Result{Title: "Channels", Columns: []string{"name", "display", "id"}}
	for _, c := range channels {
		res.Rows = append(res.Rows, output.Row{"name": c.Name, "display": c.DisplayName, "id": c.Id})
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}
