package cmd

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
)

func newSearchCmd(outputMode *string) *cobra.Command {
	var teamName string
	var full bool
	var columns string
	var format string
	var limit int
	var page int
	var sinceFlag string
	var beforeFlag string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search messages (server-side; supports Mattermost search modifiers)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runSearch(app, strings.Join(args, " "), teamName, full, columns, format, limit, page, sinceFlag, beforeFlag, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&teamName, "team", "", "team to search within (defaults to your team if you have only one)")
	cmd.Flags().BoolVar(&full, "full", false, "show full message text instead of a single-line preview")
	cmd.Flags().StringVar(&columns, "columns", "", "columns to show (e.g. time,user,message or -permalink)")
	cmd.Flags().StringVar(&format, "format", "", "output format: table|tree")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results (default 60)")
	cmd.Flags().IntVar(&page, "page", 0, "page number (0-based)")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "only posts after this time (duration like 24h or RFC3339)")
	cmd.Flags().StringVar(&beforeFlag, "before", "", "only posts before this time (RFC3339)")
	registerTeamFlagCompletion(cmd)
	return cmd
}

func runSearch(app *appContext, query, teamName string, full bool, columns, format string, limit, page int, sinceFlag, beforeFlag string, w io.Writer) error {
	ctx := context.Background()
	teamID, resolvedTeam, err := app.resolveTeam(ctx, teamName)
	if err != nil {
		return err
	}
	if sinceFlag != "" {
		if t, err := parseSince(sinceFlag); err == nil {
			query += " after:" + time.UnixMilli(t).UTC().Format("2006-01-02")
		}
	}
	if beforeFlag != "" {
		if t, err := time.Parse("2006-01-02", beforeFlag); err == nil {
			query += " before:" + t.Format("2006-01-02")
		}
	}
	pl, err := app.api.Search(ctx, teamID, query, false, limit, page)
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
	res := renderMessages(ctx, app, "Search results", postsInOrder(pl), resolvedTeam, full, cols, false)
	return app.renderWith(w, res, format)
}

// postsInOrder returns the posts of a PostList in the server-provided Order
// (relevance/recency for search), skipping missing entries.
func postsInOrder(pl *model.PostList) []*model.Post {
	if pl == nil {
		return nil
	}
	posts := make([]*model.Post, 0, len(pl.Order))
	for _, id := range pl.Order {
		if p := pl.Posts[id]; p != nil {
			posts = append(posts, p)
		}
	}
	return posts
}
