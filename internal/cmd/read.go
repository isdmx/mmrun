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
	team   string
	full   bool
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
	cmd.Flags().StringVar(&opts.team, "team", "", "team for resolving a bare channel name (defaults to your team if you have only one)")
	cmd.Flags().BoolVar(&opts.full, "full", false, "show full message text instead of a single-line preview")
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
	permalinkTeam := ""

	switch {
	case opts.thread != "":
		pl, err = app.api.PostThread(ctx, opts.thread)
		title = "Thread"
	default:
		ch, rerr := app.resolveChannel(ctx, channelRef, opts.team)
		if rerr != nil {
			return rerr
		}
		permalinkTeam = permalinkTeamFor(ctx, app, ch)
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

	res := renderMessages(ctx, app, title, chronological(pl), permalinkTeam, opts.full)
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}

// permalinkTeamFor returns the team name usable in a permalink for a channel,
// or "" when the channel has no team (e.g. a direct message).
func permalinkTeamFor(ctx context.Context, app *appContext, ch *model.Channel) string {
	if ch == nil || ch.TeamId == "" {
		return ""
	}
	if t, err := app.api.Team(ctx, ch.TeamId); err == nil && t != nil {
		return t.Name
	}
	return ""
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
