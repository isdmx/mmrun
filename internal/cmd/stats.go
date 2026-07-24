package cmd

import (
	"context"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newStatsCmd(outputMode *string) *cobra.Command {
	return &cobra.Command{
		Use:               "stats <channel>",
		Short:             "Show channel statistics",
		Example:           "  mmrun stats python\n  mmrun stats '~town-square'",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeChannelArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runStats(app, args[0], cmd.OutOrStdout())
		},
	}
}

func runStats(app *appContext, channelRef string, w io.Writer) error {
	ctx := context.Background()
	ch, err := app.resolveChannel(ctx, channelRef, "")
	if err != nil {
		return err
	}
	s, err := app.api.ChannelStats(ctx, ch.Id)
	if err != nil {
		return err
	}
	res := output.Result{
		Title: "Channel stats", Columns: []string{"field", "value"},
		Rows: []output.Row{
			{"field": "members", "value": strconv.FormatInt(s.MemberCount, 10)},
			{"field": "pinned", "value": strconv.FormatInt(s.PinnedPostCount, 10)},
		},
	}
	return app.render(w, res)
}
