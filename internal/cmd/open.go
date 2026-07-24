package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newOpenCmd(outputMode *string) *cobra.Command {
	return &cobra.Command{
		Use:               "open <id>",
		Short:             "Open a post or channel in the browser",
		Example:           "  mmrun open <post-id>\n  mmrun open <channel-id>",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePostIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			url, err := resolveOpenURL(context.Background(), app, args[0])
			if err != nil {
				return err
			}
			res := output.Result{Text: url}
			if rerr := app.render(cmd.OutOrStdout(), res); rerr != nil {
				return rerr
			}
			return openBrowser(context.Background(), url)
		},
	}
}

func resolveOpenURL(ctx context.Context, app *appContext, id string) (string, error) {
	server := strings.TrimRight(app.api.ServerURL(), "/")

	ch, cerr := app.api.Channel(ctx, id)
	if cerr == nil && ch != nil && ch.TeamId != "" {
		team, terr := app.api.Team(ctx, ch.TeamId)
		if terr == nil && team != nil {
			return fmt.Sprintf("%s/%s/channels/%s", server, team.Name, ch.Name), nil
		}
	}

	pl, perr := app.api.PostThread(ctx, id)
	if perr == nil && pl != nil {
		if root, ok := pl.Posts[id]; ok && root != nil {
			ch, cerr := app.api.Channel(ctx, root.ChannelId)
			if cerr == nil && ch != nil && ch.TeamId != "" {
				team, terr := app.api.Team(ctx, ch.TeamId)
				if terr == nil && team != nil {
					return fmt.Sprintf("%s/%s/pl/%s", server, team.Name, id), nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not resolve %q as post or channel", id)
}
