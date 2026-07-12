package cmd

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

type threadListOpts struct {
	team   string
	unread bool
	limit  int
	full   bool
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
	cmd.Flags().IntVar(&opts.limit, "limit", 30, "maximum number of threads to fetch")
	cmd.Flags().BoolVar(&opts.full, "full", false, "show full root message text instead of a single-line preview")
}

var threadColumns = []string{"last_reply", "channel", "user", "replies", "unread", "post_id", "permalink", "message"}

func runThreadList(app *appContext, opts threadListOpts, w io.Writer) error {
	ctx := context.Background()
	teamID, teamName, err := app.resolveTeam(ctx, opts.team)
	if err != nil {
		return err
	}
	threads, err := app.api.UserThreads(ctx, app.userID, teamID, opts.unread, opts.limit)
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

	res := output.Result{Title: "Followed threads", Columns: threadColumns}
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
				msg = preview(msg, maxMessagePreview)
			}
			row := output.Row{
				"last_reply": time.UnixMilli(tr.LastReplyAt).Format(time.RFC3339),
				"channel":    channelLabel(ctx, app, p.ChannelId, channelNames),
				"user":       user,
				"replies":    strconv.FormatInt(tr.ReplyCount, 10),
				"unread":     strconv.FormatInt(tr.UnreadReplies, 10),
				"post_id":    p.Id,
				"message":    msg,
			}
			if server != "" && teamName != "" {
				row["permalink"] = server + "/" + teamName + "/pl/" + p.Id
			}
			res.Rows = append(res.Rows, row)
		}
	}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}
