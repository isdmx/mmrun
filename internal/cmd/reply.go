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

//nolint:gocognit // stdin batch pattern inlined per command per task spec
func newReplyCmd(outputMode *string) *cobra.Command {
	var opts postOpts
	var replyNoStdin bool
	cmd := &cobra.Command{
		Use:               "reply <post-id> <message>",
		Short:             "Reply to a post in its channel",
		Example:           "  mmrun reply <post-id> 'great idea'",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: completePostIDArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && !replyNoStdin && isStdinPipe() {
				if args[0] == "-" {
					return fmt.Errorf("cannot use '-' for message text when reading targets from stdin; pass the message explicitly")
				}
				targets, serr := readStdinTargets()
				if serr != nil {
					return serr
				}
				if len(targets) == 0 {
					return fmt.Errorf("no targets on stdin")
				}
				app, aerr := requireSession(*outputMode)
				if aerr != nil {
					return aerr
				}
				msg := args[0]
				success, failed := 0, 0
				for _, id := range targets {
					if perr := runReply(app, id, msg, opts, cmd.OutOrStdout()); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: reply %s: %v\n", id, perr)
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
			if len(args) == 0 {
				return fmt.Errorf("requires a post-id argument or piped input and a message")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runReply(app, args[0], args[1], opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringArrayVar(&opts.files, "file", nil, "path to a file to attach (repeatable)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "resolve and preview without posting")
	cmd.Flags().BoolVar(&replyNoStdin, "no-stdin", false, "read post ID from positional arg even when piped")
	return cmd
}

func runReply(app *appContext, postID, message string, opts postOpts, w io.Writer) error {
	ctx := context.Background()
	pl, err := app.api.PostThread(ctx, postID)
	if err != nil {
		return fmt.Errorf("resolve post %q: %w", postID, err)
	}
	target, ok := pl.Posts[postID]
	if !ok || target == nil {
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
				{"field": "channel", "value": target.ChannelId},
				{"field": "reply_to", "value": postID},
				{"field": "files", "value": strings.Join(opts.files, ", ")},
				{"field": "message", "value": msg},
			},
		}
		return app.render(w, res)
	}

	fileIDs, err := uploadFiles(ctx, app, target.ChannelId, opts.files)
	if err != nil {
		return err
	}
	rootID := target.RootId
	if rootID == "" {
		rootID = postID
	}
	created, err := app.api.CreatePost(ctx, &model.Post{
		ChannelId: target.ChannelId,
		Message:   msg,
		RootId:    rootID,
		FileIds:   fileIDs,
	})
	if err != nil {
		return err
	}
	return app.render(w, output.Result{Text: created.Id})
}
