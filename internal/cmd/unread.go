package cmd

import (
	"context"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newUnreadCmd(outputMode *string) *cobra.Command {
	var teamName string
	cmd := &cobra.Command{
		Use:   "unread",
		Short: "List channels with unread messages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runUnread(app, teamName, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&teamName, "team", "", "team")
	registerTeamFlagCompletion(cmd)
	return cmd
}

func runUnread(app *appContext, teamName string, w io.Writer) error {
	ctx := context.Background()
	teamID, _, err := app.resolveTeam(ctx, teamName)
	if err != nil {
		return err
	}
	channels, err := app.api.ChannelsForUser(ctx, teamID, app.userID)
	if err != nil {
		return err
	}
	res := output.Result{Title: "Unread", Columns: []string{"channel", "mentions", "messages"}}
	for _, c := range channels {
		u, uerr := app.api.ChannelUnread(ctx, c.Id, app.userID)
		if uerr == nil && u != nil && u.MsgCount > 0 {
			label := c.DisplayName
			if label == "" {
				label = c.Name
			}
			res.Rows = append(res.Rows, output.Row{
				"channel":  label,
				"mentions": strconv.FormatInt(u.MentionCount, 10),
				"messages": strconv.FormatInt(u.MsgCount, 10),
			})
		}
	}
	return app.render(w, res)
}
