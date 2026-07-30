package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

//nolint:gocognit,gocyclo,funlen // stdin batch pattern inlined per command per task spec
func newPinCmd(outputMode *string) *cobra.Command {
	pin := &cobra.Command{Use: "pin", Short: "Pin and unpin posts"}

	var noStdin bool
	pinPost := &cobra.Command{
		Use:   "add <post-id>",
		Short: "Pin a post",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !noStdin && isStdinPipe() {
				targets, serr := readStdinTargets()
				if serr != nil {
					return serr
				}
				if len(targets) == 0 {
					return fmt.Errorf("no targets on stdin")
				}
				app, err := requireSession(*outputMode)
				if err != nil {
					return err
				}
				success, failed := 0, 0
				for _, id := range targets {
					if perr := app.api.PinPost(context.Background(), id); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: pin %s: %v\n", id, perr)
						failed++
					} else {
						fmt.Fprintf(os.Stderr, "pinned %s\n", id)
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
				return fmt.Errorf("requires a post-id argument or piped input")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runPin(app, args[0], cmd.OutOrStdout())
		},
	}
	pinPost.Flags().BoolVar(&noStdin, "no-stdin", false, "read post ID from positional arg even when piped")
	pinPost.ValidArgsFunction = completePostIDArg
	pin.AddCommand(pinPost)

	var yes bool
	var noStdinUnpin bool
	unpin := &cobra.Command{
		Use:   "remove <post-id>",
		Short: "Unpin a post (requires --yes)",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("unpin requires --yes to confirm")
			}
			if len(args) == 0 && !noStdinUnpin && isStdinPipe() {
				targets, serr := readStdinTargets()
				if serr != nil {
					return serr
				}
				if len(targets) == 0 {
					return fmt.Errorf("no targets on stdin")
				}
				app, err := requireSession(*outputMode)
				if err != nil {
					return err
				}
				success, failed := 0, 0
				for _, id := range targets {
					if perr := app.api.UnpinPost(context.Background(), id); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: unpin %s: %v\n", id, perr)
						failed++
					} else {
						fmt.Fprintf(os.Stderr, "unpinned %s\n", id)
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
				return fmt.Errorf("requires a post-id argument or piped input")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runUnpin(app, args[0], yes, cmd.OutOrStdout())
		},
	}
	unpin.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	unpin.Flags().BoolVar(&noStdinUnpin, "no-stdin", false, "read post ID from positional arg even when piped")
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
