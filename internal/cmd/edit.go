package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newEditCmd(outputMode *string) *cobra.Command {
	edit := &cobra.Command{Use: "edit", Short: "Edit and delete posts"}

	edit.AddCommand(&cobra.Command{
		Use:   "edit <post-id> <msg>",
		Short: "Edit the text of a post",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runEdit(app, args[0], args[1], cmd.OutOrStdout())
		},
	})

	var yes bool
	del := &cobra.Command{
		Use:   "delete <post-id>",
		Short: "Delete a post (requires --yes)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("delete requires --yes to confirm")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runDelete(app, args[0], yes, cmd.OutOrStdout())
		},
	}
	del.Flags().BoolVar(&yes, "yes", false, "confirm deletion")
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
