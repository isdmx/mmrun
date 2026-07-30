package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

//nolint:gocognit,gocyclo,funlen // stdin batch pattern inlined per command per task spec
func newReactCmd(outputMode *string) *cobra.Command {
	react := &cobra.Command{
		Use: "react", Short: "Manage reactions",
		Example: "  mmrun react add <post-id> :rocket:\n  mmrun react remove <post-id> :rocket: --yes",
	}

	var addNoStdin bool
	add := &cobra.Command{
		Use:   "add <post-id> <emoji>",
		Short: "Add a reaction to a post",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && !addNoStdin && isStdinPipe() {
				targets, serr := readStdinTargets()
				if serr != nil {
					return serr
				}
				if len(targets) == 0 {
					return fmt.Errorf("no targets on stdin")
				}
				app, appErr := requireSession(*outputMode)
				if appErr != nil {
					return appErr
				}
				emoji := cleanEmoji(args[0])
				success, failed := 0, 0
				for _, id := range targets {
					if perr := app.api.SaveReaction(context.Background(), id, app.userID, emoji); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: react %s: %v\n", id, perr)
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
				return fmt.Errorf("requires a post-id argument or piped input and an emoji")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runReact(app, args[0], args[1], cmd.OutOrStdout())
		},
	}
	add.Flags().BoolVar(&addNoStdin, "no-stdin", false, "read post ID from positional arg even when piped")
	add.ValidArgsFunction = completePostIDArg
	react.AddCommand(add)

	var yes bool
	var unreactNoStdin bool
	unreact := &cobra.Command{
		Use:   "remove <post-id> <emoji>",
		Short: "Remove your reaction from a post (requires --yes)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("remove requires --yes to confirm")
			}
			if len(args) == 1 && !unreactNoStdin && isStdinPipe() {
				targets, serr := readStdinTargets()
				if serr != nil {
					return serr
				}
				if len(targets) == 0 {
					return fmt.Errorf("no targets on stdin")
				}
				app, appErr := requireSession(*outputMode)
				if appErr != nil {
					return appErr
				}
				emoji := cleanEmoji(args[0])
				success, failed := 0, 0
				for _, id := range targets {
					if perr := app.api.DeleteReaction(context.Background(), id, app.userID, emoji); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: unreact %s: %v\n", id, perr)
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
				return fmt.Errorf("requires a post-id argument or piped input and an emoji")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runUnreact(app, args[0], args[1], yes, cmd.OutOrStdout())
		},
	}
	unreact.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	unreact.Flags().BoolVar(&unreactNoStdin, "no-stdin", false, "read post ID from positional arg even when piped")
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
