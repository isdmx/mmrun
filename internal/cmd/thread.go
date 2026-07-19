package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

type threadListOpts struct {
	team       string
	unread     bool
	limit      int
	full       bool
	columns    string
	noMarkdown bool
}

func newThreadCmd(outputMode *string) *cobra.Command {
	thread := &cobra.Command{
		Use:   "thread",
		Short: "List and read followed threads",
		Args:  cobra.NoArgs,
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
	threadRead := &cobra.Command{
		Use:   "read <post-id>",
		Short: "Read a thread and optionally mark it as read",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runThreadRead(app, args[0], markRead, format, "", "", !noMarkdown, cmd.OutOrStdout())
		},
	}
	threadRead.Flags().BoolVar(&markRead, "mark-read", false, "mark the thread as read")
	threadRead.Flags().StringVar(&format, "format", "", "output format: table|tree")
	threadRead.Flags().StringVar(&style, "style", "", "output style: table|chat|tree (default from config)")
	threadRead.Flags().StringVar(&timeFormat, "time-format", "", "timestamp format: rfc3339|relative")
	threadRead.Flags().BoolVar(&noMarkdown, "no-markdown", false, "disable markdown rendering")
	threadRead.ValidArgsFunction = completePostIDArg
	thread.AddCommand(threadRead)
	return thread
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
	cmd.Flags().IntVar(&opts.limit, "limit", 0, "maximum number of threads to fetch (default from config or 50)")
	cmd.Flags().BoolVar(&opts.full, "full", false, "show full root message text instead of a single-line preview")
	cmd.Flags().StringVar(&opts.columns, "columns", "", "columns to show (e.g. user,replies,message)")
	cmd.Flags().BoolVar(&opts.noMarkdown, "no-markdown", false, "disable markdown rendering")
}

var threadColumns = []string{"last_reply", "channel", "user", "replies", "unread", "files", "post_id", "permalink", "message"}

func runThreadList(app *appContext, opts threadListOpts, w io.Writer) error {
	ctx := context.Background()
	teamID, teamName, err := app.resolveTeam(ctx, opts.team)
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
	threads, err := app.api.UserThreads(ctx, app.userID, teamID, opts.unread, limit)
	if err != nil {
		return err
	}

	// Collect root posts to batch-resolve authors and channel names.
	var roots []*model.Post
	if threads != nil {
		for _, tr := range threads.Threads {
			if tr != nil && tr.Post != nil {
				roots = append(roots, tr.Post)
			}
		}
	}
	usernames := resolveUsernames(ctx, app, roots)
	channelNames := map[string]string{}
	clean := app.outputMode != "json" && !opts.full
	server := serverBase(app)

	res := output.Result{Title: "Followed threads", Columns: cols}
	if threads != nil {
		for _, tr := range threads.Threads {
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
			if server != "" && teamName != "" {
				row["permalink"] = server + "/" + teamName + "/pl/" + p.Id
			}
			res.Rows = append(res.Rows, row)
		}
	}
	return app.renderOpts(w, res, "", "", "", !opts.noMarkdown)
}

func runThreadRead(app *appContext, postID string, markRead bool, format, style, timeFormat string, markdown bool, w io.Writer) error {
	ctx := context.Background()
	pl, err := app.api.PostThread(ctx, postID)
	if err != nil {
		return err
	}
	res := renderMessages(ctx, app, "Thread", chronological(pl), "", true, messageColumns, true)
	if aerr := app.renderOpts(w, res, format, style, timeFormat, markdown); aerr != nil {
		return aerr
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
