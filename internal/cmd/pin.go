package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newPinCmd(outputMode *string) *cobra.Command {
	pin := &cobra.Command{Use: "pin", Short: "Pin and unpin posts"}

	pinPost := &cobra.Command{
		Use:   "add <post-id>",
		Short: "Pin a post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runPin(app, args[0], cmd.OutOrStdout())
		},
	}
	pinPost.ValidArgsFunction = completePostIDArg
	pin.AddCommand(pinPost)

	var yes bool
	unpin := &cobra.Command{
		Use:   "remove <post-id>",
		Short: "Unpin a post (requires --yes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("unpin requires --yes to confirm")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runUnpin(app, args[0], yes, cmd.OutOrStdout())
		},
	}
	unpin.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	unpin.ValidArgsFunction = completePostIDArg
	pin.AddCommand(unpin)

	return pin
}

func runPin(app *appContext, postID string, w io.Writer) error {
	ctx := context.Background()
	if err := app.api.PinPost(ctx, postID); err != nil {
		return err
	}
	res := output.Result{Text: "pinned " + postID}
	return app.render(w, res)
}

func runUnpin(app *appContext, postID string, yes bool, w io.Writer) error {
	if !yes {
		return fmt.Errorf("unpin requires --yes to confirm")
	}
	ctx := context.Background()
	if err := app.api.UnpinPost(ctx, postID); err != nil {
		return err
	}
	res := output.Result{Text: "unpinned " + postID}
	return app.render(w, res)
}
