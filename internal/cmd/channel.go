package cmd

import (
	"context"
	"io"
	"sort"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	"github.com/isdmx/mmrun/internal/output"
)

func newChannelCmd(outputMode *string) *cobra.Command {
	channel := &cobra.Command{
		Use:     "channel",
		Short:   "List and search channels",
		Example: "  mmrun channel --team sberdevices\n  mmrun channel search town --team sberdevices",
		Args:    cobra.NoArgs,
	}
	// Bare `channel` (and `channel list`) list channels.
	addChannelListRun(channel, outputMode)

	list := &cobra.Command{
		Use:   "list",
		Short: "List channels you belong to (direct messages hidden by default)",
		Args:  cobra.NoArgs,
	}
	addChannelListRun(list, outputMode)

	var searchTeam string
	search := &cobra.Command{
		Use:   "search <term>",
		Short: "Search channels by name (including ones you have not joined)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireSession(*outputMode)
			if err != nil {
				return err
			}
			return runChannelSearch(app, searchTeam, strings.Join(args, " "), cmd.OutOrStdout())
		},
	}
	search.Flags().StringVar(&searchTeam, "team", "", "team to search within (defaults to your team if you have only one)")
	registerTeamFlagCompletion(search)

	channel.AddCommand(list, search)
	return channel
}

// addChannelListRun wires a command to run the channel-list action with --team
// and --type flags. It is used for both the bare `channel` command and its
// explicit `list` subcommand.
func addChannelListRun(cmd *cobra.Command, outputMode *string) {
	var team, chType string
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		app, err := requireSession(*outputMode)
		if err != nil {
			return err
		}
		return runChannelList(app, team, chType, cmd.OutOrStdout())
	}
	cmd.Flags().StringVar(&team, "team", "", "team name (defaults to your team if you have only one)")
	cmd.Flags().StringVar(&chType, "type", "default", "filter: default|public|private|dm|group|all")
	registerTeamFlagCompletion(cmd)
}

func runChannelList(app *appContext, teamName, chType string, w io.Writer) error {
	ctx := context.Background()
	teams, err := app.resolveTeams(ctx, teamName)
	if err != nil {
		return err
	}
	var channels []*model.Channel
	seen := map[string]bool{}
	for _, t := range teams {
		chs, cerr := app.api.ChannelsForUser(ctx, t.id, app.userID)
		if cerr != nil {
			if teamName != "" {
				return cerr
			}
			continue
		}
		for _, c := range chs {
			if !seen[c.Id] {
				seen[c.Id] = true
				channels = append(channels, c)
			}
		}
	}

	includeDM := chType == "dm" || chType == "all"
	var dmLabels map[string]string
	if includeDM {
		dmLabels = directChannelLabels(ctx, app, channels)
	}
	includeGroup := chType == "group" || chType == "all"
	var groupLabels map[string]string
	if includeGroup {
		groupLabels = groupChannelLabels(ctx, app, channels)
	}

	res := output.Result{Title: "Channels", Columns: []string{"type", "name", "display", "id"}}
	for _, c := range channels {
		if !matchChannelType(c.Type, chType) {
			continue
		}
		name, display := c.Name, c.DisplayName
		switch c.Type {
		case model.ChannelTypeDirect:
			if other := dmLabels[c.Id]; other != "" {
				name, display = other, other
			}
		case model.ChannelTypeGroup:
			if label := groupLabels[c.Id]; label != "" {
				name, display = label, label
			}
		default:
			// other channel types keep their raw name and display name
		}
		res.Rows = append(res.Rows, output.Row{
			"type":    channelTypeLabel(c.Type),
			"name":    name,
			"display": display,
			"id":      c.Id,
		})
	}
	return app.render(w, res)
}

func runChannelSearch(app *appContext, teamName, term string, w io.Writer) error {
	ctx := context.Background()
	teams, err := app.resolveTeams(ctx, teamName)
	if err != nil {
		return err
	}
	var channels []*model.Channel
	seen := map[string]bool{}
	for _, t := range teams {
		chs, cerr := app.api.SearchChannels(ctx, t.id, term)
		if cerr != nil {
			if teamName != "" {
				return cerr
			}
			continue
		}
		for _, c := range chs {
			if !seen[c.Id] {
				seen[c.Id] = true
				channels = append(channels, c)
			}
		}
	}
	res := output.Result{Title: "Channels", Columns: []string{"type", "name", "display", "id"}}
	for _, c := range channels {
		res.Rows = append(res.Rows, output.Row{
			"type":    channelTypeLabel(c.Type),
			"name":    c.Name,
			"display": c.DisplayName,
			"id":      c.Id,
		})
	}
	return app.render(w, res)
}

// matchChannelType reports whether a channel of the given type should be shown
// for the requested filter. The "default" filter shows named channels (public,
// private, group) but hides direct messages.
func matchChannelType(t model.ChannelType, filter string) bool {
	switch filter {
	case "all":
		return true
	case "public":
		return t == model.ChannelTypeOpen
	case "private":
		return t == model.ChannelTypePrivate
	case "dm":
		return t == model.ChannelTypeDirect
	case "group":
		return t == model.ChannelTypeGroup
	default: // "default"
		return t != model.ChannelTypeDirect
	}
}

func channelTypeLabel(t model.ChannelType) string {
	switch t {
	case model.ChannelTypeOpen:
		return "public"
	case model.ChannelTypePrivate:
		return "private"
	case model.ChannelTypeDirect:
		return "dm"
	case model.ChannelTypeGroup:
		return "group"
	default:
		return string(t)
	}
}

// directChannelLabels maps each direct-message channel ID to the other
// participant's username, resolved via a single batched user lookup.
func directChannelLabels(ctx context.Context, app *appContext, channels []*model.Channel) map[string]string {
	otherByChannel := map[string]string{}
	seen := map[string]bool{}
	var ids []string
	for _, c := range channels {
		if c.Type != model.ChannelTypeDirect {
			continue
		}
		var other string
		for _, part := range strings.Split(c.Name, "__") {
			if part != app.userID {
				other = part
			}
		}
		if other == "" {
			continue
		}
		otherByChannel[c.Id] = other
		if !seen[other] {
			seen[other] = true
			ids = append(ids, other)
		}
	}
	users, err := app.api.UsersByIDs(ctx, ids)
	if err != nil {
		return map[string]string{}
	}
	username := map[string]string{}
	for _, u := range users {
		username[u.Id] = u.Username
	}
	labels := map[string]string{}
	for chID, otherID := range otherByChannel {
		if n := username[otherID]; n != "" {
			label := "@" + n
			for _, bid := range app.botIDs {
				if bid == otherID {
					label = "🤖" + label
					break
				}
			}
			labels[chID] = label
		}
	}
	return labels
}

// groupChannelLabels maps each group-message channel ID to a comma-separated
// list of the other participants' @usernames ("you" for a self-only GM).
// Member lists come from per-channel ChannelMembers calls; usernames resolve
// via a single batched UsersByIDs call.
func groupChannelLabels(ctx context.Context, app *appContext, channels []*model.Channel) map[string]string {
	othersByChannel := map[string][]string{}
	seen := map[string]bool{}
	var ids []string
	for _, c := range channels {
		if c.Type != model.ChannelTypeGroup {
			continue
		}
		members, err := app.api.ChannelMembers(ctx, c.Id, 0, 200)
		if err != nil {
			continue
		}
		others := make([]string, 0, len(members))
		for _, m := range members {
			if m.UserId == app.userID {
				continue
			}
			others = append(others, m.UserId)
			if !seen[m.UserId] {
				seen[m.UserId] = true
				ids = append(ids, m.UserId)
			}
		}
		sort.Strings(others)
		othersByChannel[c.Id] = others
	}
	users, err := app.api.UsersByIDs(ctx, ids)
	if err != nil {
		return map[string]string{}
	}
	username := map[string]string{}
	for _, u := range users {
		username[u.Id] = u.Username
	}
	labels := map[string]string{}
	for chID, others := range othersByChannel {
		if len(others) == 0 {
			labels[chID] = "you"
			continue
		}
		parts := make([]string, 0, len(others))
		for _, id := range others {
			if n := username[id]; n != "" {
				parts = append(parts, "@"+n)
			}
		}
		if len(parts) > 0 {
			labels[chID] = strings.Join(parts, ", ")
		}
	}
	return labels
}
