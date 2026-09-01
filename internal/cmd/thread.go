package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/isdmx/mmrun/internal/output"
)

type threadListOpts struct {
	team       string
	unread     bool
	limit      int
	full       bool
	columns    string
	noMarkdown bool
	quiet      bool
	since      string
}

func newThreadCmd(outputMode *string) *cobra.Command {
	thread := &cobra.Command{
		Use:     "thread",
		Short:   "List and read followed threads",
		Example: "  mmrun thread --unread --limit 10\n  mmrun thread read <post-id> --mark-read",
		Args:    cobra.NoArgs,
	}
	addThreadListRun(thread, outputMode)

	list := &cobra.Command{
		Use:   "list",
		Short: "List the threads you follow, most recently updated first",
		Args:  cobra.NoArgs,
	}
	addThreadListRun(list, outputMode)

	thread.AddCommand(list)

	var markRead bool
	var format string
	var style string
	var timeFormat string
	var noMarkdown bool
	var noStdinRead bool
	var sinceFlag string
	var team string
	var limit int
	threadRead := &cobra.Command{
		Use:   "read <post-id>",
		Short: "Read a thread and optionally mark it as read",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThreadReadCmd(outputMode, args, sinceFlag, team, limit, markRead, noMarkdown, noStdinRead, format, style, timeFormat, cmd.OutOrStdout())
		},
	}
	threadRead.Flags().BoolVar(&markRead, "mark-read", false, "mark the thread as read")
	threadRead.Flags().StringVar(&format, "format", "", "output format: table|tree")
	threadRead.Flags().StringVar(&style, "style", "", "output style: table|chat|tree (default from config)")
	threadRead.Flags().StringVar(&timeFormat, "time-format", "", "timestamp format: rfc3339|relative")
	threadRead.Flags().BoolVar(&noMarkdown, "no-markdown", false, "disable markdown rendering")
	threadRead.Flags().BoolVar(&noStdinRead, "no-stdin", false, "read post ID from positional arg even when piped")
	threadRead.Flags().StringVar(&sinceFlag, "since", "", "only posts after this time (duration like 24h, RFC3339, or date like 2026-09-01); without a post-id lists recent messages across followed threads")
	threadRead.Flags().StringVar(&team, "team", "", "team to list threads for (only used without a post-id)")
	threadRead.Flags().IntVar(&limit, "limit", 0, "max total messages (only used without a post-id)")
	registerTeamFlagCompletion(threadRead)
	threadRead.ValidArgsFunction = completePostIDArg
	thread.AddCommand(threadRead)

	follow := &cobra.Command{
		Use:   "follow <post-id>",
		Short: "Follow a thread",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThreadFollowCmd(outputMode, args, true, false, cmd.OutOrStdout())
		},
	}
	follow.ValidArgsFunction = completePostIDArg
	thread.AddCommand(follow)

	var unfollowYes bool
	unfollow := &cobra.Command{
		Use:   "unfollow <post-id>",
		Short: "Unfollow a thread (requires --yes)",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThreadFollowCmd(outputMode, args, false, unfollowYes, cmd.OutOrStdout())
		},
	}
	unfollow.Flags().BoolVar(&unfollowYes, "yes", false, "confirm unfollow")
	unfollow.ValidArgsFunction = completePostIDArg
	thread.AddCommand(unfollow)

	return thread
}

func runThreadReadCmd(outputMode *string, args []string, sinceFlag, team string, limit int, markRead, noMarkdown, noStdinRead bool, format, style, timeFormat string, w io.Writer) error {
	since, err := parseThreadSince(sinceFlag)
	if err != nil {
		return err
	}
	if len(args) == 0 && since > 0 {
		app, err := requireSession(*outputMode)
		if err != nil {
			return err
		}
		return runThreadReadRecent(app, team, since, limit, format, style, timeFormat, !noMarkdown, w)
	}
	if len(args) == 0 && !noStdinRead && isStdinPipe() {
		return runThreadReadStdin(outputMode, since, markRead, noMarkdown, format, w)
	}
	if len(args) == 0 {
		return fmt.Errorf("requires a post-id argument or piped input")
	}
	app, err := requireSession(*outputMode)
	if err != nil {
		return err
	}
	return runThreadRead(app, args[0], since, markRead, format, "", "", !noMarkdown, w)
}

func runThreadReadStdin(outputMode *string, since int64, markRead, noMarkdown bool, format string, w io.Writer) error {
	targets, err := readStdinTargets()
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no targets on stdin")
	}
	app, err := requireSession(*outputMode)
	if err != nil {
		return err
	}
	success, failed := 0, 0
	for _, id := range targets {
		if perr := runThreadRead(app, id, since, markRead, format, "", "", !noMarkdown, w); perr != nil {
			fmt.Fprintf(os.Stderr, "mmrun: thread %s: %v\n", id, perr)
			failed++
		} else {
			success++
		}
	}
	if success == 0 {
		return fmt.Errorf("all %d targets failed", len(targets))
	}
	if failed > 0 {
		return ErrPartialSuccess
	}
	return nil
}

func runThreadFollowCmd(outputMode *string, args []string, follow, yes bool, w io.Writer) error {
	if !follow && !yes {
		return fmt.Errorf("unfollow requires --yes to confirm")
	}
	if len(args) == 0 {
		return fmt.Errorf("requires a post-id argument")
	}
	app, err := requireSession(*outputMode)
	if err != nil {
		return err
	}
	return runThreadFollow(app, args[0], follow, w)
}

func runThreadFollow(app *appContext, postID string, follow bool, w io.Writer) error {
	ctx := context.Background()
	pl, err := app.api.PostThread(ctx, postID)
	if err != nil {
		return err
	}
	target := pl.Posts[postID]
	if target == nil {
		return fmt.Errorf("post %q not found", postID)
	}
	rootID := target.RootId
	if rootID == "" {
		rootID = postID
	}
	root := pl.Posts[rootID]
	if root == nil {
		return fmt.Errorf("thread root %q not found", rootID)
	}
	ch, err := app.api.Channel(ctx, root.ChannelId)
	if err != nil || ch == nil || ch.TeamId == "" {
		return fmt.Errorf("cannot resolve team for thread %q", rootID)
	}
	if follow {
		if err := app.api.FollowThread(ctx, app.userID, ch.TeamId, rootID); err != nil {
			return err
		}
		return app.render(w, output.Result{Text: "followed thread " + rootID})
	}
	if err := app.api.UnfollowThread(ctx, app.userID, ch.TeamId, rootID); err != nil {
		return err
	}
	return app.render(w, output.Result{Text: "unfollowed thread " + rootID})
}

func addThreadListRun(cmd *cobra.Command, outputMode *string) {
	var opts threadListOpts
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		app, err := requireSession(*outputMode)
		if err != nil {
			return err
		}
		return runThreadList(app, opts, cmd.OutOrStdout())
	}
	cmd.Flags().StringVar(&opts.team, "team", "", "team to list threads for (defaults to your team if you have only one)")
	cmd.Flags().BoolVar(&opts.unread, "unread", false, "only threads with unread replies")
	cmd.Flags().StringVar(&opts.since, "since", "", "only threads with activity on/after this time (duration like 24h, RFC3339, or date like 2026-09-01)")
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "maximum number of threads to fetch (default from config or 50)")
	cmd.Flags().BoolVar(&opts.full, "full", false, "show full root message text instead of a single-line preview")
	cmd.Flags().StringVar(&opts.columns, "columns", "", "columns to show (e.g. user,replies,message)")
	cmd.Flags().BoolVar(&opts.noMarkdown, "no-markdown", false, "disable markdown rendering")
	cmd.Flags().BoolVarP(&opts.quiet, "quiet", "q", false, "output only post IDs, one per line")
	registerTeamFlagCompletion(cmd)
}

var threadColumns = []string{"last_reply", "channel", "user", "replies", "unread", "files", "post_id", "permalink", "message"}

func runThreadList(app *appContext, opts threadListOpts, w io.Writer) error {
	ctx := context.Background()
	since, err := parseThreadSince(opts.since)
	if err != nil {
		return err
	}
	teams, err := app.resolveTeams(ctx, opts.team)
	if err != nil {
		return err
	}
	limit := opts.limit
	if limit <= 0 {
		limit = app.defaultLimit
	}
	cols, err := resolveColumns(threadColumns, opts.columns)
	if err != nil {
		return err
	}

	// Fetch threads across teams, deduping team-less threads (DMs/GMs) that
	// surface under every team.
	var merged []*model.ThreadResponse
	teamNames := map[string]string{}
	seen := map[string]bool{}
	for _, t := range teams {
		threads, terr := app.api.UserThreads(ctx, app.userID, t.id, opts.unread, limit, since)
		if terr != nil {
			if opts.team != "" {
				return terr
			}
			continue
		}
		filterThreadsSince(threads, since)
		if threads == nil {
			continue
		}
		for _, tr := range threads.Threads {
			if tr == nil {
				continue
			}
			id := tr.PostId
			if id == "" && tr.Post != nil {
				id = tr.Post.Id
			}
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			teamNames[id] = t.name
			merged = append(merged, tr)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].LastReplyAt > merged[j].LastReplyAt })
	if limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}

	clean := app.outputMode != "json" && !opts.full
	res := buildThreadResult(ctx, app, merged, cols, clean, serverBase(app), teamNames)
	if opts.quiet {
		return output.NewWithOptions(app.outputMode, stdoutFile(w), output.Options{Quiet: true, QuietColumn: "post_id"}).Render(w, res)
	}
	return app.renderOpts(w, res, "", "", "", !opts.noMarkdown)
}

func buildThreadResult(ctx context.Context, app *appContext, threads []*model.ThreadResponse, cols []string, clean bool, server string, teamNames map[string]string) output.Result {
	usernames := resolveUsernames(ctx, app, collectThreadRoots(threads))
	channelNames := map[string]string{}
	res := output.Result{Title: "Followed threads", Columns: cols}
	for _, tr := range threads {
		if tr == nil || tr.Post == nil {
			continue
		}
		p := tr.Post
		user := usernames[p.UserId]
		if user == "" {
			user = p.UserId
		}
		msg := p.Message
		if clean {
			msg = preview(msg, app.previewLen)
		}
		row := output.Row{
			"last_reply": time.UnixMilli(tr.LastReplyAt).Format(time.RFC3339),
			"channel":    channelLabel(ctx, app, p.ChannelId, channelNames),
			"user":       user,
			"replies":    strconv.FormatInt(tr.ReplyCount, 10),
			"unread":     strconv.FormatInt(tr.UnreadReplies, 10),
			"files":      fileSummary(p),
			"post_id":    p.Id,
			"message":    msg,
		}
		if server != "" && teamNames[p.Id] != "" {
			row["permalink"] = server + "/" + teamNames[p.Id] + "/pl/" + p.Id
		}
		res.Rows = append(res.Rows, row)
	}
	return res
}

func collectThreadRoots(threads []*model.ThreadResponse) []*model.Post {
	roots := make([]*model.Post, 0, len(threads))
	for _, tr := range threads {
		if tr != nil && tr.Post != nil {
			roots = append(roots, tr.Post)
		}
	}
	return roots
}

func parseThreadSince(since string) (int64, error) {
	if since == "" {
		return 0, nil
	}
	return parseSince(since)
}

// filterThreadsSince re-filters client-side because the server's Since uses
// LastUpdateAt rather than LastReplyAt/CreateAt, so results can otherwise differ.
func filterThreadsSince(threads *model.Threads, since int64) {
	if since <= 0 || threads == nil {
		return
	}
	kept := threads.Threads[:0]
	for _, tr := range threads.Threads {
		if tr != nil && (tr.LastReplyAt >= since || (tr.Post != nil && tr.Post.CreateAt >= since)) {
			kept = append(kept, tr)
		}
	}
	threads.Threads = kept
}

func runThreadRead(app *appContext, postID string, since int64, markRead bool, format, style, timeFormat string, markdown bool, w io.Writer) error {
	ctx := context.Background()
	pl, err := app.api.PostThread(ctx, postID)
	if err != nil {
		return err
	}
	posts := client.SortPosts(pl)
	if since > 0 {
		posts = filterPostsSince(posts, since)
	}
	res := renderMessages(ctx, app, "Thread", posts, "", true, messageColumns, true, "")
	if aerr := app.renderOpts(w, res, format, style, timeFormat, markdown); aerr != nil {
		return aerr
	}
	if app.autoMarkRead {
		if root, ok := pl.Posts[postID]; ok && root != nil {
			ch, cerr := app.api.Channel(ctx, root.ChannelId)
			if cerr == nil && ch != nil && ch.TeamId != "" {
				_ = app.api.UpdateThreadRead(ctx, app.userID, ch.TeamId, postID)
			}
		}
	}
	if markRead {
		if root, ok := threadRoot(pl, postID); ok {
			ch, cerr := app.api.Channel(ctx, root.ChannelId)
			if cerr == nil && ch.TeamId != "" {
				if uerr := app.api.UpdateThreadRead(ctx, app.userID, ch.TeamId, postID); uerr != nil {
					return uerr
				}
				fmt.Fprintf(os.Stderr, "Marked thread as read.\n")
			}
		}
	}
	return nil
}

func runThreadReadRecent(app *appContext, team string, since int64, limit int, format, style, timeFormat string, markdown bool, w io.Writer) error {
	ctx := context.Background()
	teams, err := app.resolveTeams(ctx, team)
	if err != nil {
		return err
	}
	if limit <= 0 {
		limit = app.defaultLimit
	}
	rootIDs, err := activeThreadRoots(ctx, app, teams, since, team != "")
	if err != nil {
		return err
	}
	threads := fetchThreadsSince(ctx, app, rootIDs, since)

	var flat []*model.Post
	headers := map[int]string{}
	for _, t := range threads {
		if limit > 0 && len(flat) >= limit {
			break
		}
		headers[len(flat)] = threadHeaderText(app, t)
		for _, p := range t.posts {
			flat = append(flat, p)
			if limit > 0 && len(flat) >= limit {
				break
			}
		}
	}

	permalinkTeam := ""
	if len(teams) > 0 {
		permalinkTeam = teams[0].name
	}
	res := renderMessages(ctx, app, "Recent thread messages", flat, permalinkTeam, true, messageColumns, false, style)
	injectThreadHeaders(&res, headers)
	return app.renderOpts(w, res, format, style, timeFormat, markdown)
}

func activeThreadRoots(ctx context.Context, app *appContext, teams []teamRef, since int64, strict bool) ([]string, error) {
	var rootIDs []string
	seen := map[string]bool{}
	for _, t := range teams {
		threads, err := app.api.UserThreads(ctx, app.userID, t.id, false, 0, since)
		if err != nil {
			if strict {
				return nil, err
			}
			continue
		}
		if threads == nil {
			continue
		}
		for _, tr := range threads.Threads {
			if tr == nil {
				continue
			}
			id := tr.PostId
			if id == "" && tr.Post != nil {
				id = tr.Post.Id
			}
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			rootIDs = append(rootIDs, id)
		}
	}
	return rootIDs, nil
}

func filterPostsSince(posts []*model.Post, since int64) []*model.Post {
	if since <= 0 {
		return posts
	}
	kept := make([]*model.Post, 0, len(posts))
	for _, p := range posts {
		if p != nil && p.CreateAt >= since {
			kept = append(kept, p)
		}
	}
	return kept
}

type threadPosts struct {
	root  *model.Post
	posts []*model.Post
}

func fetchThreadsSince(ctx context.Context, app *appContext, rootIDs []string, since int64) []threadPosts {
	if len(rootIDs) == 0 {
		return nil
	}
	type result struct {
		t threadPosts
	}
	results := make(chan result, len(rootIDs))
	sem := make(chan struct{}, 8)
	for _, id := range rootIDs {
		sem <- struct{}{}
		go func(rootID string) {
			defer func() { <-sem }()
			var tp threadPosts
			pl, err := app.api.PostThread(ctx, rootID)
			if err != nil || pl == nil {
				results <- result{t: tp}
				return
			}
			for _, p := range pl.Posts {
				if p == nil {
					continue
				}
				if p.Id == rootID {
					tp.root = p
					continue
				}
				if p.CreateAt >= since {
					tp.posts = append(tp.posts, p)
				}
			}
			sort.Slice(tp.posts, func(i, j int) bool { return tp.posts[i].CreateAt < tp.posts[j].CreateAt })
			results <- result{t: tp}
		}(id)
	}
	var threads []threadPosts
	for range rootIDs {
		tp := (<-results).t
		if len(tp.posts) > 0 {
			threads = append(threads, tp)
		}
	}
	sort.Slice(threads, func(i, j int) bool {
		return threads[i].posts[len(threads[i].posts)-1].CreateAt > threads[j].posts[len(threads[j].posts)-1].CreateAt
	})
	return threads
}

func threadHeaderText(app *appContext, t threadPosts) string {
	msg := "thread"
	if t.root != nil {
		if m := preview(t.root.Message, app.previewLen); m != "" {
			msg = m
		}
	}
	return fmt.Sprintf("%s (%d new)", msg, len(t.posts))
}

func injectThreadHeaders(res *output.Result, headers map[int]string) {
	if len(headers) == 0 {
		return
	}
	rows := make([]output.Row, 0, len(res.Rows)+len(headers))
	for i, row := range res.Rows {
		if text, ok := headers[i]; ok {
			rows = append(rows, output.Row{"_type": "thread_header", "message": text})
		}
		rows = append(rows, row)
	}
	res.Rows = rows
}

// threadRoot returns the root post of a thread PostList, if present.
func threadRoot(pl *model.PostList, postID string) (*model.Post, bool) {
	if pl == nil {
		return nil, false
	}
	root, ok := pl.Posts[postID]
	if !ok || root == nil {
		return nil, false
	}
	return root, true
}
