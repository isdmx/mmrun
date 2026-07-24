package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newMarkReadCmd(outputMode *string) *cobra.Command {
	var typeFlag string
	cmd := &cobra.Command{
		Use:               "mark-read <id>",
		Short:             "Mark a channel or thread as read",
		Example:           "  mmrun mark-read <channel-id> --type channel\n  mmrun mark-read <post-id> --type thread",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeChannelArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runMarkRead(app, args[0], typeFlag, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&typeFlag, "type", "", "channel|thread (auto-detected if omitted)")
	return cmd
}

func runMarkRead(app *appContext, id, typ string, _ io.Writer) error {
	ctx := context.Background()
	switch strings.ToLower(typ) {
	case "channel":
		if err := app.api.ViewChannel(ctx, app.userID, id); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Marked channel as read.\n")
		return nil
	case "thread":
		return markThreadRead(ctx, app, id)
	}

	if _, err := app.api.Channel(ctx, id); err == nil {
		if verr := app.api.ViewChannel(ctx, app.userID, id); verr == nil {
			fmt.Fprintf(os.Stderr, "Marked channel as read.\n")
			return nil
		}
	}
	if err := markThreadRead(ctx, app, id); err == nil {
		return nil
	}
	return fmt.Errorf("cannot determine whether %q is a channel or thread; use --type", id)
}

// markThreadRead resolves the team for a thread via its root post's channel and
// marks the thread as read for the current user.
func markThreadRead(ctx context.Context, app *appContext, id string) error {
	pl, err := app.api.PostThread(ctx, id)
	if err != nil {
		return fmt.Errorf("resolve thread %q: %w", id, err)
	}
	root, ok := threadRoot(pl, id)
	if !ok {
		return fmt.Errorf("thread root %q not found", id)
	}
	ch, cerr := app.api.Channel(ctx, root.ChannelId)
	if cerr != nil {
		return fmt.Errorf("resolve channel %q for thread: %w", root.ChannelId, cerr)
	}
	if ch.TeamId == "" {
		return fmt.Errorf("thread %q has no team; cannot mark as read", id)
	}
	if uerr := app.api.UpdateThreadRead(ctx, app.userID, ch.TeamId, id); uerr != nil {
		return uerr
	}
	fmt.Fprintf(os.Stderr, "Marked thread as read.\n")
	return nil
}
