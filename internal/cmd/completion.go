package cmd

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"
)

// completeChannelArg is a cobra ValidArgsFunction completing the first
// positional argument with channel references (names, ~names, @usernames).
func completeChannelArg(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}
	app, err := requireSession("auto")
	if err != nil || app == nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return completeChannelCompletions(app), cobra.ShellCompDirectiveNoFileComp
}

// completePostIDArg is a cobra ValidArgsFunction completing the first
// positional argument with post IDs from the user's followed threads.
func completePostIDArg(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	app, err := requireSession("auto")
	if err != nil || app == nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return completePostIDCompletions(app), cobra.ShellCompDirectiveNoFileComp
}

// registerTeamFlagCompletion wires --team flag completion with the user's
// team names on the given command.
func registerTeamFlagCompletion(cmd *cobra.Command) {
	_ = cmd.RegisterFlagCompletionFunc("team", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		app, err := requireSession("auto")
		if err != nil || app == nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return completeTeamCompletions(app), cobra.ShellCompDirectiveNoFileComp
	})
}

// completeChannelCompletions returns shell completions for channel arguments:
// channel names (bare and ~-prefixed), display names, and @usernames for DMs.
func completeChannelCompletions(app *appContext) []string {
	ctx := context.Background()
	teamID, _, err := app.resolveTeam(ctx, "")
	if err != nil {
		teamID = ""
	}
	channels, err := app.api.ChannelsForUser(ctx, teamID, app.userID)
	if err != nil {
		return nil
	}
	var out []string
	var peers []string
	for _, c := range channels {
		if c == nil {
			continue
		}
		if c.Type == model.ChannelTypeDirect {
			if peer := dmPeer(c.Name, app.userID); peer != "" {
				peers = append(peers, peer)
			}
			continue
		}
		if c.Name != "" {
			out = append(out, c.Name, "~"+c.Name)
		}
		if c.DisplayName != "" {
			out = append(out, c.DisplayName)
		}
	}
	return append(append(out, dmCompletions(ctx, app, peers)...), resolveSelfCompletion(ctx, app))
}

// resolveSelfCompletion returns the @username form of the authenticated user
// so their own DM address is always offered as a completion.
func resolveSelfCompletion(ctx context.Context, app *appContext) string {
	if app.username != "" {
		return "@" + app.username
	}
	u, err := app.api.Me(ctx)
	if err != nil || u == nil {
		return ""
	}
	return "@" + u.Username
}

// dmCompletions maps DM peer user IDs to @username completions, falling back
// to the raw user ID when the username cannot be resolved.
func dmCompletions(ctx context.Context, app *appContext, peers []string) []string {
	if len(peers) == 0 {
		return nil
	}
	username := map[string]string{}
	if users, err := app.api.UsersByIDs(ctx, peers); err == nil {
		for _, u := range users {
			if u != nil {
				username[u.Id] = u.Username
			}
		}
	}
	out := make([]string, 0, len(peers))
	for _, p := range peers {
		if n := username[p]; n != "" {
			out = append(out, "@"+n)
		} else {
			out = append(out, "@"+p)
		}
	}
	return out
}

// dmPeer returns the other participant's user ID in a direct-message channel
// name of the form "id1__id2", or "" when there is none.
func dmPeer(name, selfID string) string {
	for _, p := range splitDMName(name) {
		if p != "" && p != selfID {
			return p
		}
	}
	return ""
}

// splitDMName splits a DM channel name on the "__" separator.
func splitDMName(name string) []string {
	res := make([]string, 0, 2)
	start := 0
	for i := 0; i+1 < len(name); i++ {
		if name[i] == '_' && name[i+1] == '_' {
			res = append(res, name[start:i])
			start = i + 2
			i++
		}
	}
	if start < len(name) {
		res = append(res, name[start:])
	}
	return res
}

// completeTeamCompletions returns shell completions for --team flags.
func completeTeamCompletions(app *appContext) []string {
	teams, err := app.api.TeamsForUser(context.Background(), app.userID)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(teams))
	for _, t := range teams {
		if t != nil && t.Name != "" {
			out = append(out, t.Name)
		}
	}
	return out
}

// completePostIDCompletions returns shell completions for post-ID arguments,
// sourced from the user's most recently active followed threads.
func completePostIDCompletions(app *appContext) []string {
	ctx := context.Background()
	teamID, _, err := app.resolveTeam(ctx, "")
	if err != nil {
		teamID = ""
	}
	threads, err := app.api.UserThreads(ctx, app.userID, teamID, false, 50)
	if err != nil || threads == nil {
		return nil
	}
	out := make([]string, 0, len(threads.Threads))
	for _, tr := range threads.Threads {
		if tr != nil && tr.PostId != "" {
			out = append(out, tr.PostId)
		}
	}
	return out
}
