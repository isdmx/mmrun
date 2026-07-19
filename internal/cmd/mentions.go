package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
)

func newMentionsCmd(outputMode *string) *cobra.Command {
	var teamName string
	var limit int
	var full bool
	var columns string
	var format string
	var style string
	var timeFormat string
	var noMarkdown bool
	cmd := &cobra.Command{
		Use:     "mentions",
		Short:   "Search posts that mention you",
		Example: "  mmrun mentions --team sberdevices --limit 20",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runMentions(app, teamName, columns, limit, full, format, style, timeFormat, !noMarkdown, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&teamName, "team", "", "restrict to this team")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results (default from config)")
	cmd.Flags().BoolVar(&full, "full", false, "show full message text")
	cmd.Flags().StringVar(&columns, "columns", "", "columns to show")
	cmd.Flags().StringVar(&format, "format", "", "output format: table|tree")
	cmd.Flags().StringVar(&style, "style", "", "output style: table|chat|tree (default from config)")
	cmd.Flags().StringVar(&timeFormat, "time-format", "", "timestamp format: rfc3339|relative")
	cmd.Flags().BoolVar(&noMarkdown, "no-markdown", false, "disable markdown rendering")
	registerTeamFlagCompletion(cmd)
	return cmd
}

func runMentions(app *appContext, teamName, columns string, limit int, full bool, format, style, timeFormat string, markdown bool, w io.Writer) error {
	ctx := context.Background()
	if app.username == "" {
		return fmt.Errorf("no username in session; re-login to store it")
	}
	term := "@" + app.username
	if limit <= 0 {
		limit = app.defaultLimit
	}

	var allPosts []*model.Post
	seen := map[string]bool{}

	collect := func(teamID string) error {
		pl, err := app.api.Search(ctx, teamID, term, false, limit, 0)
		if err != nil {
			return err
		}
		if pl != nil {
			for _, id := range pl.Order {
				if p := pl.Posts[id]; p != nil && !seen[p.Id] {
					seen[p.Id] = true
					allPosts = append(allPosts, p)
				}
			}
		}
		return nil
	}

	var permalinkTeam string
	if teamName != "" {
		id, name, err := app.resolveTeam(ctx, teamName)
		if err != nil {
			return err
		}
		permalinkTeam = name
		if err := collect(id); err != nil {
			return err
		}
	} else {
		teams, err := app.api.TeamsForUser(ctx, app.userID)
		if err != nil {
			return err
		}
		for _, t := range teams {
			_ = collect(t.Id)
			if permalinkTeam == "" {
				permalinkTeam = t.Name
			}
		}
	}

	sort.SliceStable(allPosts, func(i, j int) bool { return allPosts[i].CreateAt < allPosts[j].CreateAt })
	if len(allPosts) > limit {
		allPosts = allPosts[:limit]
	}

	spec := columns
	if spec == "" {
		spec = app.columnsDefault
	}
	cols, err := resolveColumns(messageColumns, spec)
	if err != nil {
		return err
	}
	res := renderMessages(ctx, app, "Mentions", allPosts, permalinkTeam, full, cols, false)
	return app.renderOpts(w, res, format, style, timeFormat, markdown)
}
