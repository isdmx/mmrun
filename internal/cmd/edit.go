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
func newEditCmd(outputMode *string) *cobra.Command {
	edit := &cobra.Command{
		Use: "edit", Short: "Edit and delete posts",
		Example: "  mmrun edit edit <post-id> 'new text'\n  mmrun edit delete <post-id> --yes",
	}

	var editNoStdin bool
	editPost := &cobra.Command{
		Use:   "edit <post-id> <msg>",
		Short: "Edit the text of a post",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && !editNoStdin && isStdinPipe() {
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
					if _, perr := app.api.PatchPost(context.Background(), id, msg); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: edit %s: %v\n", id, perr)
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
			return runEdit(app, args[0], args[1], cmd.OutOrStdout())
		},
	}
	editPost.Flags().BoolVar(&editNoStdin, "no-stdin", false, "read post ID from positional arg even when piped")
	editPost.ValidArgsFunction = completePostIDArg
	edit.AddCommand(editPost)

	var yes bool
	var noStdin bool
	del := &cobra.Command{
		Use:   "delete <post-id>",
		Short: "Delete a post (requires --yes)",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("delete requires --yes to confirm")
			}
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
					if perr := app.api.DeletePost(context.Background(), id); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: delete %s: %v\n", id, perr)
						failed++
					} else {
						fmt.Fprintf(os.Stderr, "deleted %s\n", id)
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
			return runDelete(app, args[0], yes, cmd.OutOrStdout())
		},
	}
	del.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
	del.Flags().BoolVar(&noStdin, "no-stdin", false, "read post ID from positional arg even when piped")
	del.ValidArgsFunction = completePostIDArg
	edit.AddCommand(del)

	return edit
}

func runEdit(app *appContext, postID, msg string, w io.Writer) error {
	ctx := context.Background()
	p, err := app.api.PatchPost(ctx, postID, msg)
	if err != nil {
		return err
	}
	res := output.Result{Text: p.Id}
	return app.render(w, res)
}

func runDelete(app *appContext, postID string, yes bool, w io.Writer) error {
	if !yes {
		return fmt.Errorf("delete requires --yes to confirm")
	}
	ctx := context.Background()
	if err := app.api.DeletePost(ctx, postID); err != nil {
		return err
	}
	res := output.Result{Text: "deleted " + postID}
	return app.render(w, res)
}
