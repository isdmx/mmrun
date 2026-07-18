package cmd

import (
	"context"
	"io"

	"github.com/spf13/cobra"
)

func newPinnedCmd(outputMode *string) *cobra.Command {
	return &cobra.Command{
		Use:   "pinned <channel>",
		Short: "List pinned posts in a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runPinned(app, args[0], cmd.OutOrStdout())
		},
	}
}

func runPinned(app *appContext, channelRef string, w io.Writer) error {
	ctx := context.Background()
	ch, err := app.resolveChannel(ctx, channelRef, "")
	if err != nil {
		return err
	}
	pl, err := app.api.PinnedPosts(ctx, ch.Id)
	if err != nil {
		return err
	}
	res := renderMessages(ctx, app, "Pinned", chronological(pl), "", true, messageColumns, true)
	return app.render(w, res)
}
