package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/dmitriev/mmrun/internal/output"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
)

type readOpts struct {
	limit  int
	since  string
	thread string
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
	cmd.Flags().StringVar(&opts.since, "since", "", "only messages since this time: a duration (e.g. 24h) or RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.thread, "thread", "", "fetch the thread rooted at this post ID instead of the channel")
	return cmd
}

// parseSince interprets a --since value as either a Go duration relative to now
// (e.g. "24h") or an absolute RFC3339 timestamp, returning Unix milliseconds.
func parseSince(v string) (int64, error) {
	if d, err := time.ParseDuration(v); err == nil {
		return time.Now().Add(-d).UnixMilli(), nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UnixMilli(), nil
	}
	return 0, fmt.Errorf("invalid --since %q: use a duration like 24h or an RFC3339 timestamp", v)
}

func runRead(app *appContext, channelRef string, opts readOpts, w io.Writer) error {
	ctx := context.Background()

	var pl *model.PostList
	var err error
	title := "Messages"

	switch {
	case opts.thread != "":
		pl, err = app.api.PostThread(ctx, opts.thread)
		title = "Thread"
	default:
		ch, rerr := app.api.ResolveChannel(ctx, channelRef, app.defaultTeam, app.userID)
		if rerr != nil {
			return rerr
		}
		if opts.since != "" {
			since, perr := parseSince(opts.since)
			if perr != nil {
				return perr
			}
			pl, err = app.api.PostsSince(ctx, ch.Id, since)
		} else {
			pl, err = app.api.PostsForChannel(ctx, ch.Id, opts.limit)
		}
	}
	if err != nil {
		return err
	}

	res := output.Result{Title: title, Columns: []string{"time", "user_id", "message"}}
	for _, p := range chronological(pl) {
		res.Rows = append(res.Rows, output.Row{
			"time":    time.UnixMilli(p.CreateAt).Format(time.RFC3339),
			"user_id": p.UserId,
			"message": p.Message,
		})
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}

// chronological returns the posts of a PostList sorted oldest-first. It is
// resilient to a nil list and to Order/Posts inconsistencies.
func chronological(pl *model.PostList) []*model.Post {
	if pl == nil {
		return nil
	}
	posts := make([]*model.Post, 0, len(pl.Posts))
	for _, p := range pl.Posts {
		if p != nil {
			posts = append(posts, p)
		}
	}
	sort.SliceStable(posts, func(i, j int) bool { return posts[i].CreateAt < posts[j].CreateAt })
	return posts
}
