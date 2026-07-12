package cmd

import (
	"context"
	"io"
	"time"

	"github.com/dmitriev/mmrun/internal/output"
	"github.com/spf13/cobra"
)

type readOpts struct {
	limit int
}

func newReadCmd(outputMode *string) *cobra.Command {
	var opts readOpts
	cmd := &cobra.Command{
		Use:   "read <channel>",
		Short: "Fetch recent messages from a channel or DM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runRead(app, args[0], opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().IntVar(&opts.limit, "limit", 50, "number of messages to fetch")
	return cmd
}

func runRead(app *appContext, channelRef string, opts readOpts, w io.Writer) error {
	ctx := context.Background()
	ch, err := app.api.ResolveChannel(ctx, channelRef, app.defaultTeam)
	if err != nil {
		return err
	}
	pl, err := app.api.PostsForChannel(ctx, ch.Id, opts.limit)
	if err != nil {
		return err
	}
	res := output.Result{Title: "Messages", Columns: []string{"time", "user_id", "message"}}
	// PostList.Order is newest-first; reverse for chronological output.
	for i := len(pl.Order) - 1; i >= 0; i-- {
		p := pl.Posts[pl.Order[i]]
		if p == nil {
			continue
		}
		res.Rows = append(res.Rows, output.Row{
			"time":    time.UnixMilli(p.CreateAt).Format(time.RFC3339),
			"user_id": p.UserId,
			"message": p.Message,
		})
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}
