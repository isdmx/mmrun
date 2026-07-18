package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/isdmx/mmrun/internal/output"
)

type postOpts struct {
	replyTo string
	files   []string
	team    string
	dryRun  bool
	editor  bool
}

func newPostCmd(outputMode *string) *cobra.Command {
	var opts postOpts
	cmd := &cobra.Command{
		Use:   "post <channel> [message]",
		Short: "Post a message to a channel or DM",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			messageText := ""
			if len(args) >= 2 {
				messageText = args[1]
			}
			return runPost(app, args[0], messageText, opts, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&opts.replyTo, "reply-to", "", "root post ID to reply in-thread")
	cmd.Flags().StringArrayVar(&opts.files, "file", nil, "path to a file to attach (repeatable)")
	cmd.Flags().StringVar(&opts.team, "team", "", "team for resolving a bare channel name (defaults to your team if you have only one)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "resolve the target and preview without posting")
	cmd.Flags().BoolVar(&opts.editor, "editor", false, "open $EDITOR for the message")
	cmd.ValidArgsFunction = completeChannelArg
	return cmd
}

func runPost(app *appContext, channelRef, argsMessage string, opts postOpts, w io.Writer) error {
	ctx := context.Background()
	msg := argsMessage
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	if argsMessage == "" && isTTY {
		var err error
		msg, err = editorMessage(ctx, argsMessage)
		if err != nil {
			return err
		}
	}
	if msg == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		msg = string(data)
	}
	if opts.editor {
		var err error
		msg, err = editorMessage(ctx, msg)
		if err != nil {
			return err
		}
	}
	ch, err := app.resolveChannel(ctx, channelRef, opts.team)
	if err != nil {
		return err
	}
	if opts.dryRun {
		res := output.Result{
			Title:   "Dry run (not sent)",
			Columns: []string{"field", "value"},
			Rows: []output.Row{
				{"field": "channel", "value": ch.Id},
				{"field": "reply_to", "value": opts.replyTo},
				{"field": "files", "value": strings.Join(opts.files, ", ")},
				{"field": "message", "value": msg},
			},
		}
		return app.render(w, res)
	}
	fileIDs, err := uploadFiles(ctx, app, ch.Id, opts.files)
	if err != nil {
		return err
	}
	post := &model.Post{ChannelId: ch.Id, Message: msg, RootId: opts.replyTo, FileIds: fileIDs}
	created, err := app.api.CreatePost(ctx, post)
	if err != nil {
		return err
	}
	res := output.Result{Text: created.Id}
	return app.render(w, res)
}

// editorMessage opens $EDITOR (falling back to $VISUAL, then vim) on a temp
// file pre-filled with the given text and returns the edited content.
func editorMessage(ctx context.Context, prefill string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vim"
	}
	f, err := os.CreateTemp("", "mmrun-msg-*.md")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if prefill != "" {
		if _, err := f.WriteString(prefill); err != nil {
			_ = f.Close()
			return "", err
		}
	}
	_ = f.Close()
	cmd := exec.CommandContext(ctx, editor, f.Name()) //nolint:gosec // launching the user's own $EDITOR is the feature
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q: %w", editor, err)
	}
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", err
	}
	return string(data), nil
}
