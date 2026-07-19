package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newReplyCmd(outputMode *string) *cobra.Command {
	var opts postOpts
	cmd := &cobra.Command{
		Use:     "reply <post-id> <message>",
		Short:   "Reply to a post in its channel",
		Example: "  mmrun reply <post-id> 'great idea'",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runReply(app, args[0], args[1], opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringArrayVar(&opts.files, "file", nil, "path to a file to attach (repeatable)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "resolve and preview without posting")
	return cmd
}

func runReply(app *appContext, postID, message string, opts postOpts, w io.Writer) error {
	ctx := context.Background()
	pl, err := app.api.PostThread(ctx, postID)
	if err != nil {
		return fmt.Errorf("resolve post %q: %w", postID, err)
	}
	root, ok := pl.Posts[postID]
	if !ok || root == nil {
		return fmt.Errorf("post %q not found", postID)
	}

	msg := message
	if msg == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		msg = string(data)
	}

	if opts.dryRun {
		res := output.Result{
			Title:   "Dry run (not sent)",
			Columns: []string{"field", "value"},
			Rows: []output.Row{
				{"field": "channel", "value": root.ChannelId},
				{"field": "reply_to", "value": postID},
				{"field": "files", "value": strings.Join(opts.files, ", ")},
				{"field": "message", "value": msg},
			},
		}
		return app.render(w, res)
	}

	fileIDs, err := uploadFiles(ctx, app, root.ChannelId, opts.files)
	if err != nil {
		return err
	}
	created, err := app.api.CreatePost(ctx, &model.Post{
		ChannelId: root.ChannelId,
		Message:   msg,
		RootId:    postID,
		FileIds:   fileIDs,
	})
	if err != nil {
		return err
	}
	return app.render(w, output.Result{Text: created.Id})
}
