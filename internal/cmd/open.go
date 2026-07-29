package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

//nolint:gocognit // stdin batch pattern inlined per command per task spec
func newOpenCmd(outputMode *string) *cobra.Command {
	var noStdin bool
	cmd := &cobra.Command{
		Use:               "open <id>",
		Short:             "Open a post or channel in the browser",
		Example:           "  mmrun open <post-id>\n  mmrun open <channel-id>",
		Args:              cobra.RangeArgs(0, 1),
		ValidArgsFunction: completePostIDArg,
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
					url, uerr := resolveOpenURL(context.Background(), app, id)
					if uerr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: open %s: %v\n", id, uerr)
						failed++
						continue
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), url)
					if oerr := openBrowser(context.Background(), url); oerr != nil {
						fmt.Fprintf(os.Stderr, "mmrun: open %s: %v\n", id, oerr)
						failed++
						continue
					}
					success++
				}
				if success == 0 {
					return fmt.Errorf("all %d targets failed", len(targets))
				}
				if failed > 0 {
					return ErrPartialSuccess
				}
				return nil
			}
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			url, err := resolveOpenURL(context.Background(), app, args[0])
			if err != nil {
				return err
			}
			res := output.Result{Text: url}
			if rerr := app.render(cmd.OutOrStdout(), res); rerr != nil {
				return rerr
			}
			return openBrowser(context.Background(), url)
		},
	}
	cmd.Flags().BoolVar(&noStdin, "no-stdin", false, "read post ID from positional arg even when piped")
	return cmd
}

func resolveOpenURL(ctx context.Context, app *appContext, id string) (string, error) {
	server := strings.TrimRight(app.api.ServerURL(), "/")

	ch, cerr := app.api.Channel(ctx, id)
	if cerr == nil && ch != nil && ch.TeamId != "" {
		team, terr := app.api.Team(ctx, ch.TeamId)
		if terr == nil && team != nil {
			return fmt.Sprintf("%s/%s/channels/%s", server, team.Name, ch.Name), nil
		}
	}

	pl, perr := app.api.PostThread(ctx, id)
	if perr == nil && pl != nil {
		if root, ok := pl.Posts[id]; ok && root != nil {
			ch, cerr := app.api.Channel(ctx, root.ChannelId)
			if cerr == nil && ch != nil && ch.TeamId != "" {
				team, terr := app.api.Team(ctx, ch.TeamId)
				if terr == nil && team != nil {
					return fmt.Sprintf("%s/%s/pl/%s", server, team.Name, id), nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not resolve %q as post or channel", id)
}
