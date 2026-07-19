package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newReactCmd(outputMode *string) *cobra.Command {
	react := &cobra.Command{
		Use: "react", Short: "Manage reactions",
		Example: "  mmrun react add <post-id> :rocket:\n  mmrun react remove <post-id> :rocket: --yes",
	}

	add := &cobra.Command{
		Use:   "add <post-id> <emoji>",
		Short: "Add a reaction to a post",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runReact(app, args[0], args[1], cmd.OutOrStdout())
		},
	}
	add.ValidArgsFunction = completePostIDArg
	react.AddCommand(add)

	var yes bool
	unreact := &cobra.Command{
		Use:   "remove <post-id> <emoji>",
		Short: "Remove your reaction from a post (requires --yes)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runUnreact(app, args[0], args[1], yes, cmd.OutOrStdout())
		},
	}
	unreact.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	unreact.ValidArgsFunction = completePostIDArg
	react.AddCommand(unreact)

	return react
}

func cleanEmoji(e string) string { return strings.Trim(e, ":") }

func runReact(app *appContext, postID, emoji string, w io.Writer) error {
	ctx := context.Background()
	emoji = cleanEmoji(emoji)
	if err := app.api.SaveReaction(ctx, postID, app.userID, emoji); err != nil {
		return err
	}
	res := output.Result{Text: "reacted :" + emoji + ": on " + postID}
	return app.render(w, res)
}

func runUnreact(app *appContext, postID, emoji string, yes bool, w io.Writer) error {
	if !yes {
		return fmt.Errorf("remove requires --yes to confirm")
	}
	ctx := context.Background()
	emoji = cleanEmoji(emoji)
	if err := app.api.DeleteReaction(ctx, postID, app.userID, emoji); err != nil {
		return err
	}
	res := output.Result{Text: "removed :" + emoji + ": from " + postID}
	return app.render(w, res)
}
