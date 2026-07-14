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
		Use:   "mark-read <id>",
		Short: "Mark a channel or thread as read",
		Args:  cobra.ExactArgs(1),
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
	teamID := ""
	if err == nil {
		if root, ok := threadRoot(pl, id); ok {
			if ch, cerr := app.api.Channel(ctx, root.ChannelId); cerr == nil {
				teamID = ch.TeamId
			}
		}
	}
	if uerr := app.api.UpdateThreadRead(ctx, app.userID, teamID, id); uerr != nil {
		return uerr
	}
	fmt.Fprintf(os.Stderr, "Marked thread as read.\n")
	return nil
}
