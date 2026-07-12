package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

type postOpts struct {
	replyTo string
	file    string
	team    string
}

func newPostCmd(outputMode *string) *cobra.Command {
	var opts postOpts
	cmd := &cobra.Command{
		Use:   "post <channel> <message>",
		Short: "Post a message to a channel or DM",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runPost(app, args[0], args[1], opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.replyTo, "reply-to", "", "root post ID to reply in-thread")
	cmd.Flags().StringVar(&opts.file, "file", "", "path to a file to attach")
	cmd.Flags().StringVar(&opts.team, "team", "", "team for resolving a bare channel name (defaults to your team if you have only one)")
	return cmd
}

func runPost(app *appContext, channelRef, message string, opts postOpts, w io.Writer) error {
	ctx := context.Background()
	ch, err := app.resolveChannel(ctx, channelRef, opts.team)
	if err != nil {
		return err
	}
	post := &model.Post{ChannelId: ch.Id, Message: message, RootId: opts.replyTo}

	if opts.file != "" {
		data, err := os.ReadFile(opts.file)
		if err != nil {
			return err
		}
		resp, err := app.api.UploadFile(ctx, data, ch.Id, filepath.Base(opts.file))
		if err != nil {
			return err
		}
		for _, fi := range resp.FileInfos {
			post.FileIds = append(post.FileIds, fi.Id)
		}
	}

	created, err := app.api.CreatePost(ctx, post)
	if err != nil {
		return err
	}
	res := output.Result{Text: created.Id}
	return output.New(app.outputMode, stdoutFile(w)).Render(w, res)
}
