package cmd

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/dmitriev/mmrun/internal/output"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
)

func newTailCmd(outputMode *string) *cobra.Command {
	var team string
	cmd := &cobra.Command{
		Use:   "tail <channel>",
		Short: "Stream new messages from a channel until interrupted",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			return runTail(ctx, app, args[0], team, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "team for resolving a bare channel name (defaults to your team if you have only one)")
	return cmd
}

// postedEventToRow converts a websocket event into an output Row, returning
// false for events that are not posts in the target channel.
func postedEventToRow(channelID, event string, data map[string]any) (output.Row, bool) {
	if event != "posted" {
		return nil, false
	}
	raw, ok := data["post"].(string)
	if !ok {
		return nil, false
	}
	var p model.Post
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, false
	}
	if channelID != "" && p.ChannelId != "" && p.ChannelId != channelID {
		return nil, false
	}
	return output.Row{
		"time":    time.UnixMilli(p.CreateAt).Format(time.RFC3339),
		"user_id": p.UserId,
		"message": p.Message,
	}, true
}

func runTail(ctx context.Context, app *appContext, channelRef, team string, w io.Writer) error {
	ch, err := app.resolveChannel(ctx, channelRef, team)
	if err != nil {
		return err
	}
	events, errs, err := app.api.StreamPosts(ctx)
	if err != nil {
		return err
	}
	r := output.New(app.outputMode, stdoutFile(w))
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			row, ok := postedEventToRow(ch.Id, ev.Event, ev.Data)
			if !ok {
				continue
			}
			_ = r.Render(w, output.Result{Columns: []string{"time", "user_id", "message"}, Rows: []output.Row{row}})
		case err := <-errs:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}
