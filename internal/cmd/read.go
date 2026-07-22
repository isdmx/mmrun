package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
)

type readOpts struct {
	limit         int
	since         string
	thread        string
	team          string
	full          bool
	columns       string
	markRead      bool
	format        string
	style         string
	timeFormat    string
	threadsOnly   bool
	tail          bool
	unreadSummary bool
	noMarkdown    bool
	links         bool
}

func newReadCmd(outputMode *string) *cobra.Command {
	var opts readOpts
	cmd := &cobra.Command{
		Use:     "read <channel>",
		Short:   "Fetch recent messages from a channel or DM",
		Example: "  mmrun read python --limit 20\n  mmrun read '~town-square' --style chat --since 24h\n  mmrun read @alice --full",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if !cmd.Flags().Changed("full") {
				opts.full = app.full
			}
			if !cmd.Flags().Changed("full") {
				opts.full = app.full
			}
			if err != nil {
				return err
			}
			return runRead(app, args[0], opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "number of messages to fetch (default from config or 50)")
	cmd.Flags().StringVar(&opts.since, "since", "", "only messages since this time: a duration (e.g. 24h) or RFC3339 timestamp")
	cmd.Flags().StringVar(&opts.thread, "thread", "", "fetch the thread rooted at this post ID instead of the channel")
	cmd.Flags().StringVar(&opts.team, "team", "", "team for resolving a bare channel name (defaults to your team if you have only one)")
	cmd.Flags().BoolVar(&opts.full, "full", false, "show full message text instead of a single-line preview")
	cmd.Flags().StringVar(&opts.columns, "columns", "", "columns to show (e.g. time,user,message or -permalink)")
	cmd.Flags().BoolVar(&opts.markRead, "mark-read", false, "mark the channel as read after fetching messages")
	cmd.Flags().StringVar(&opts.format, "format", "", "output format: table|tree (default from config)")
	cmd.Flags().StringVar(&opts.style, "style", "", "output style: table|chat|tree (default from config)")
	cmd.Flags().StringVar(&opts.timeFormat, "time-format", "", "timestamp format: rfc3339|relative")
	cmd.Flags().BoolVar(&opts.threadsOnly, "threads-only", false, "show only root posts (no replies)")
	cmd.Flags().BoolVar(&opts.tail, "tail", false, "enter live-stream mode after fetching messages")
	cmd.Flags().BoolVar(&opts.unreadSummary, "unread-summary", false, "show unread/mention counts after fetching")
	cmd.Flags().BoolVar(&opts.noMarkdown, "no-markdown", false, "disable markdown rendering")
	cmd.Flags().BoolVar(&opts.links, "links", false, "extract and list URLs from message bodies")
	cmd.ValidArgsFunction = completeChannelArg
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

	limit := opts.limit
	if limit <= 0 {
		limit = app.defaultLimit
	}

	spec := opts.columns
	if spec == "" {
		spec = app.columnsDefault
	}
	columns, err := resolveColumns(messageColumns, spec)
	if err != nil {
		return err
	}

	var pl *model.PostList
	var ch *model.Channel
	title := "Messages"
	permalinkTeam := ""
	var markCh *model.Channel

	switch {
	case opts.thread != "":
		pl, err = app.api.PostThread(ctx, opts.thread)
		title = "Thread"
	default:
		var rerr error
		ch, rerr = app.resolveChannel(ctx, channelRef, opts.team)
		if rerr != nil {
			return rerr
		}
		markCh = ch
		permalinkTeam = permalinkTeamFor(ctx, app, ch)
		if opts.since != "" {
			since, perr := parseSince(opts.since)
			if perr != nil {
				return perr
			}
			pl, err = app.api.PostsSince(ctx, ch.Id, since)
		} else {
			pl, err = app.api.PostsForChannel(ctx, ch.Id, limit)
		}
	}
	if err != nil {
		return err
	}

	posts := chronological(pl)
	if opts.threadsOnly {
		posts = filterRoots(posts)
	}

	if opts.links {
		return app.render(w, renderLinks(posts))
	}

	res := renderMessages(ctx, app, title, posts, permalinkTeam, opts.full, columns, true)
	aerr := app.renderOpts(w, res, opts.format, opts.style, opts.timeFormat, !opts.noMarkdown)
	if opts.markRead && markCh != nil {
		if herr := app.api.ViewChannel(ctx, app.userID, markCh.Id); herr != nil {
			return herr
		}
		fmt.Fprintf(os.Stderr, "Marked %s as read.\n", markCh.Name)
	}
	if opts.unreadSummary {
		unread, err := app.api.ChannelUnread(ctx, ch.Id, app.userID)
		if err == nil && unread != nil {
			if unread.MsgCount > 0 {
				fmt.Fprintf(os.Stderr, "%d unread, %d mentions\n", unread.MsgCount, unread.MentionCount)
			} else {
				fmt.Fprintln(os.Stderr, "all read")
			}
		}
	}
	if opts.tail {
		tctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		if terr := runTail(tctx, app, channelRef, opts.team, false, "", w); terr != nil {
			fmt.Fprintf(os.Stderr, "tail: %v\n", terr)
		}
	}
	return aerr
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
