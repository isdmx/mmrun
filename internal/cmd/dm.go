package cmd

import (
	"context"
	"io"
	"sort"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newDMCmd(outputMode *string) *cobra.Command {
	dm := &cobra.Command{
		Use:     "dm",
		Short:   "List direct and group messages",
		Example: "  mmrun dm list --since 24h\n  mmrun dm --since 2026-09-01 --limit 20",
		Args:    cobra.NoArgs,
	}
	// Bare `dm` (and `dm list`) list recent direct and group messages.
	addDMListRun(dm, outputMode)

	list := &cobra.Command{
		Use:   "list",
		Short: "List recent direct and group messages",
		Args:  cobra.NoArgs,
	}
	addDMListRun(list, outputMode)

	dm.AddCommand(list)
	return dm
}

// addDMListRun wires a command to run the DM-list action with --since and
// --team flags. It is used for both the bare `dm` command and its explicit
// `list` subcommand.
func addDMListRun(cmd *cobra.Command, outputMode *string) {
	var since, team, columns, format, style, timeFormat string
	var limit int
	var full, noMarkdown, links, quiet bool
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		app, err := requireSession(*outputMode)
		if err != nil {
			return err
		}
		if !cmd.Flags().Changed("full") {
			full = app.full
		}
		opts := dmOpts{
			since:      since,
			team:       team,
			limit:      limit,
			full:       full,
			columns:    columns,
			format:     format,
			style:      style,
			timeFormat: timeFormat,
			markdown:   !noMarkdown,
			links:      links,
			quiet:      quiet,
		}
		return runDMList(app, opts, cmd.OutOrStdout())
	}
	cmd.Flags().StringVar(&since, "since", "", "only messages after this time (duration like 24h, RFC3339, or date like 2026-09-01; default 24h)")
	cmd.Flags().StringVar(&team, "team", "", "restrict to this team")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results (default from config)")
	cmd.Flags().BoolVar(&full, "full", false, "show full message text instead of a single-line preview")
	cmd.Flags().StringVar(&columns, "columns", "", "columns to show (e.g. time,user,message or -permalink)")
	cmd.Flags().StringVar(&format, "format", "", "output format: table|tree")
	cmd.Flags().StringVar(&style, "style", "", "output style: table|chat|tree (default from config)")
	cmd.Flags().StringVar(&timeFormat, "time-format", "", "timestamp format: rfc3339|relative")
	cmd.Flags().BoolVar(&noMarkdown, "no-markdown", false, "disable markdown rendering")
	cmd.Flags().BoolVar(&links, "links", false, "extract and list URLs from message bodies")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "output only post IDs, one per line")
	registerTeamFlagCompletion(cmd)
}

type dmOpts struct {
	since      string
	team       string
	limit      int
	full       bool
	columns    string
	format     string
	style      string
	timeFormat string
	markdown   bool
	links      bool
	quiet      bool
}

// runDMList lists recent direct and group messages. It uses one server-side
// search per team (no per-channel round trips) and filters the results
// client-side to DM/GM channels.
func runDMList(app *appContext, opts dmOpts, w io.Writer) error {
	ctx := context.Background()

	sinceMs, err := dmSinceMs(opts.since)
	if err != nil {
		return err
	}
	// Mattermost's after: modifier is exclusive of the given day, so back
	// up one day to make --since inclusive ("on or after").
	afterTerm := " after:" + time.UnixMilli(sinceMs).UTC().AddDate(0, 0, -1).Format("2006-01-02")

	limit := opts.limit
	if limit <= 0 {
		limit = app.defaultLimit
	}

	teams, err := app.resolveTeams(ctx, opts.team)
	if err != nil {
		return err
	}
	dmChannels, err := dmChannelSet(ctx, app, teams, opts.team != "")
	if err != nil {
		return err
	}
	allPosts, permalinkTeam, err := searchDMPosts(ctx, app, teams, afterTerm, limit, dmChannels, opts.team != "")
	if err != nil {
		return err
	}

	sort.SliceStable(allPosts, func(i, j int) bool { return allPosts[i].CreateAt < allPosts[j].CreateAt })
	if len(allPosts) > limit {
		allPosts = allPosts[:limit]
	}

	spec := opts.columns
	if spec == "" {
		spec = app.columnsDefault
	}
	cols, err := resolveColumns(messageColumns, spec)
	if err != nil {
		return err
	}
	if opts.links {
		return app.render(w, renderLinks(allPosts))
	}
	res := renderMessages(ctx, app, "Direct messages", allPosts, permalinkTeam, opts.full, cols, false, opts.style)
	if opts.quiet {
		return output.NewWithOptions(app.outputMode, stdoutFile(w), output.Options{Quiet: true, QuietColumn: "post_id"}).Render(w, res)
	}
	return app.renderOpts(w, res, opts.format, opts.style, opts.timeFormat, opts.markdown)
}

func dmSinceMs(since string) (int64, error) {
	if since == "" {
		return time.Now().Add(-24 * time.Hour).UnixMilli(), nil
	}
	return parseSince(since)
}

func dmChannelSet(ctx context.Context, app *appContext, teams []teamRef, strict bool) (map[string]bool, error) {
	dmChannels := map[string]bool{}
	for _, t := range teams {
		channels, err := app.api.ChannelsForUser(ctx, t.id, app.userID)
		if err != nil {
			if strict {
				return nil, err
			}
			continue
		}
		for _, c := range channels {
			if c.Type == model.ChannelTypeDirect || c.Type == model.ChannelTypeGroup {
				dmChannels[c.Id] = true
			}
		}
	}
	return dmChannels, nil
}

func searchDMPosts(ctx context.Context, app *appContext, teams []teamRef, afterTerm string, limit int, dmChannels map[string]bool, strict bool) ([]*model.Post, string, error) {
	var allPosts []*model.Post
	seen := map[string]bool{}
	permalinkTeam := ""
	for _, t := range teams {
		pl, err := app.api.Search(ctx, t.id, afterTerm, false, limit, 0)
		if err != nil {
			if strict {
				return nil, "", err
			}
			continue
		}
		for _, p := range postsInOrder(pl) {
			if p == nil || !dmChannels[p.ChannelId] || seen[p.Id] {
				continue
			}
			seen[p.Id] = true
			allPosts = append(allPosts, p)
		}
		if permalinkTeam == "" {
			permalinkTeam = t.name
		}
	}
	return allPosts, permalinkTeam, nil
}
