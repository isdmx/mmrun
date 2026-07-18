package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newFlaggedCmd(outputMode *string) *cobra.Command {
	var team string
	var limit int
	cmd := &cobra.Command{
		Use:   "flagged",
		Short: "List posts you flagged",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runFlagged(app, team, limit, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "restrict to this team")
	cmd.Flags().IntVar(&limit, "limit", 50, "max results")
	return cmd
}

func runFlagged(app *appContext, teamName string, limit int, w io.Writer) error {
	ctx := context.Background()
	teamID, resolvedTeam, err := app.resolveTeam(ctx, teamName)
	if err != nil {
		return err
	}
	pl, err := app.api.FlaggedPosts(ctx, app.userID, teamID, 0, limit)
	if err != nil {
		return err
	}
	res := renderMessages(ctx, app, "Flagged", postsInOrder(pl), resolvedTeam, true, messageColumns, false)
	return app.render(w, res)
}

func newFlagCmd(outputMode *string) *cobra.Command {
	flag := &cobra.Command{Use: "flag", Short: "Flag and unflag posts"}

	flag.AddCommand(&cobra.Command{
		Use: "add <post-id>", Short: "Flag a post", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return app.api.FlagPost(context.Background(), args[0])
		},
	})

	var yes bool
	remove := &cobra.Command{
		Use: "remove <post-id> --yes", Short: "Unflag a post", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("unflag requires --yes to confirm")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return app.api.UnflagPost(context.Background(), args[0])
		},
	}
	remove.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	flag.AddCommand(remove)
	return flag
}
