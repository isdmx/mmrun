package cmd

import (
	"context"
	"io"

	"github.com/spf13/cobra"
)

func newPinnedCmd(outputMode *string) *cobra.Command {
	var columns string
	var full bool
	var style string
	var timeFormat string
	var format string
	var noMarkdown bool
	cmd := &cobra.Command{
		Use:     "pinned <channel>",
		Short:   "List pinned posts in a channel",
		Example: "  mmrun pinned python\n  mmrun pinned '~general' --columns time,user,message",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runPinned(app, args[0], columns, full, style, timeFormat, format, !noMarkdown, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&columns, "columns", "", "columns to show")
	cmd.Flags().BoolVar(&full, "full", false, "show full message text")
	cmd.Flags().StringVar(&style, "style", "", "output style: table|chat|tree")
	cmd.Flags().StringVar(&timeFormat, "time-format", "", "timestamp format: rfc3339|relative")
	cmd.Flags().StringVar(&format, "format", "", "output format: table|tree")
	cmd.Flags().BoolVar(&noMarkdown, "no-markdown", false, "disable markdown rendering")
	return cmd
}

func runPinned(app *appContext, channelRef, columns string, full bool, style, timeFormat, format string, markdown bool, w io.Writer) error {
	ctx := context.Background()
	ch, err := app.resolveChannel(ctx, channelRef, "")
	if err != nil {
		return err
	}
	pl, err := app.api.PinnedPosts(ctx, ch.Id)
	if err != nil {
		return err
	}
	spec := columns
	if spec == "" {
		spec = app.columnsDefault
	}
	cols, err := resolveColumns(messageColumns, spec)
	if err != nil {
		return err
	}
	res := renderMessages(ctx, app, "Pinned", chronological(pl), "", full, cols, true)
	return app.renderOpts(w, res, format, style, timeFormat, markdown)
}
