package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dmitriev/mmrun/internal/client"
	"github.com/dmitriev/mmrun/internal/output"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
)

func newTailCmd(outputMode *string) *cobra.Command {
	return &cobra.Command{
		Use:   "tail <channel>",
		Short: "Stream new messages from a channel until interrupted",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runTail(app, args[0])
		},
	}
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

func runTail(app *appContext, channelRef string) error {
	ctx := context.Background()
	ch, err := app.api.ResolveChannel(ctx, channelRef, app.defaultTeam)
	if err != nil {
		return err
	}
	c, ok := app.api.(*client.Client)
	if !ok {
		return fmt.Errorf("tail requires a live client")
	}
	events, errs, err := c.StreamPosts(ctx)
	if err != nil {
		return err
	}
	r := output.New(app.outputMode, os.Stdout)
	for {
		select {
		case ev := <-events:
			row, ok := postedEventToRow(ch.Id, ev.Event, ev.Data)
			if !ok {
				continue
			}
			_ = r.Render(os.Stdout, output.Result{Columns: []string{"time", "user_id", "message"}, Rows: []output.Row{row}})
		case err := <-errs:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}
