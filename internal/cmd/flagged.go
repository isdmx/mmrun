package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func newFlaggedCmd(outputMode *string) *cobra.Command {
	var team string
	var limit int
	var columns string
	var full bool
	var style string
	var timeFormat string
	var format string
	var noMarkdown bool
	cmd := &cobra.Command{
		Use:     "flagged",
		Short:   "List posts you flagged",
		Example: "  mmrun flagged --team sberdevices --limit 20\n  mmrun flag add <post-id>\n  mmrun flag remove <post-id> --yes",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := requireSession(*outputMode)
			if !cmd.Flags().Changed("full") {
				full = app.full
			}
			if err != nil {
				return err
			}
			return runFlagged(app, team, limit, columns, full, style, timeFormat, format, !noMarkdown, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&team, "team", "", "restrict to this team")
	cmd.Flags().IntVar(&limit, "limit", 50, "max results")
	cmd.Flags().StringVar(&columns, "columns", "", "columns to show")
	cmd.Flags().BoolVar(&full, "full", false, "show full message text")
	cmd.Flags().StringVar(&style, "style", "", "output style: table|chat|tree")
	cmd.Flags().StringVar(&timeFormat, "time-format", "", "timestamp format: rfc3339|relative")
	cmd.Flags().StringVar(&format, "format", "", "output format: table|tree")
	cmd.Flags().BoolVar(&noMarkdown, "no-markdown", false, "disable markdown rendering")
	return cmd
}

func runFlagged(app *appContext, teamName string, limit int, columns string, full bool, style, timeFormat, format string, markdown bool, w io.Writer) error {
	ctx := context.Background()
	teamID, resolvedTeam, err := app.resolveTeam(ctx, teamName)
	if err != nil {
		return err
	}
	pl, err := app.api.FlaggedPosts(ctx, app.userID, teamID, 0, limit)
	if err != nil {
		return err
	}
	spec := columns
	if spec == "" {
		spec = app.columnsDefault
	}
	cols, err := resolveColumns(messageColumns, spec)
	if err != nil {
		return err
	}
	res := renderMessages(ctx, app, "Flagged", postsInOrder(pl), resolvedTeam, full, cols, false, style)
	return app.renderOpts(w, res, format, style, timeFormat, markdown)
}

//nolint:gocognit,gocyclo // stdin batch pattern inlined per command per task spec
func newFlagCmd(outputMode *string) *cobra.Command {
	flag := &cobra.Command{Use: "flag", Short: "Flag and unflag posts"}

	var noStdin bool
	flagAdd := &cobra.Command{
		Use: "add <post-id>", Short: "Flag a post", Args: cobra.RangeArgs(0, 1),
		RunE: func(_ *cobra.Command, args []string) error {
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
					if perr := app.api.FlagPost(context.Background(), id); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: flag %s: %v\n", id, perr)
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
				return fmt.Errorf("requires a post-id argument or piped input")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return app.api.FlagPost(context.Background(), args[0])
		},
	}
	flagAdd.Flags().BoolVar(&noStdin, "no-stdin", false, "read post ID from positional arg even when piped")
	flag.AddCommand(flagAdd)

	var yes bool
	var noStdinRemove bool
	remove := &cobra.Command{
		Use: "remove <post-id> --yes", Short: "Unflag a post", Args: cobra.RangeArgs(0, 1),
		RunE: func(_ *cobra.Command, args []string) error {
			if !yes {
				return fmt.Errorf("unflag requires --yes to confirm")
			}
			if len(args) == 0 && !noStdinRemove && isStdinPipe() {
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
					if perr := app.api.UnflagPost(context.Background(), id); perr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: unflag %s: %v\n", id, perr)
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
				return fmt.Errorf("requires a post-id argument or piped input")
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return app.api.UnflagPost(context.Background(), args[0])
		},
	}
	remove.Flags().BoolVar(&yes, "yes", false, "confirm removal")
	remove.Flags().BoolVar(&noStdinRemove, "no-stdin", false, "read post ID from positional arg even when piped")
	flag.AddCommand(remove)
	return flag
}
