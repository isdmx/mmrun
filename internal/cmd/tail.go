package cmd

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newTailCmd(outputMode *string) *cobra.Command {
	var team string
	var mentionsOnly bool
	var fromUser string
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
			return runTail(ctx, app, args[0], team, mentionsOnly, fromUser, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "team for resolving a bare channel name (defaults to your team if you have only one)")
	cmd.Flags().BoolVar(&mentionsOnly, "mentions-only", false, "only show posts that mention you")
	cmd.Flags().StringVar(&fromUser, "from", "", "only show posts by this user")
	cmd.ValidArgsFunction = completeChannelArg
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

// postUserID extracts the UserId from a websocket post data payload.
func postUserID(data map[string]any) string {
	if data == nil {
		return ""
	}
	raw, ok := data["post"].(string)
	if !ok {
		return ""
	}
	var p model.Post
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return ""
	}
	return p.UserId
}

func runTail(ctx context.Context, app *appContext, channelRef, team string, mentionsOnly bool, fromUser string, w io.Writer) error {
	ch, err := app.resolveChannel(ctx, channelRef, team)
	if err != nil {
		return err
	}
	events, errs, err := app.api.StreamPosts(ctx)
	if err != nil {
		return err
	}
	// Resolve fromUser to ID once before entering the loop.
	var fromUserID string
	if fromUser != "" {
		if u, uerr := app.api.UserByUsername(ctx, fromUser); uerr == nil && u != nil {
			fromUserID = u.Id
		}
	}
	r := output.NewWithOptions(app.outputMode, stdoutFile(w), output.Options{Color: app.color})
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
			if mentionsOnly && !strings.Contains(row["message"], "@"+app.username) {
				continue
			}
			if fromUserID != "" {
				if postUserID(ev.Data) != fromUserID {
					continue
				}
			}
			_ = r.Render(w, output.Result{Columns: []string{"time", "user_id", "message"}, Rows: []output.Row{row}})
		case err := <-errs:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}
