package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/isdmx/mmrun/internal/output"
)

const maxMessagePreview = 140

var messageColumns = []string{"time", "channel", "user", "files", "root_id", "post_id", "permalink", "message"}

// renderMessages builds a message Result from posts in the given order. It
// resolves user IDs to usernames (one batched call), channel IDs to readable
// names (cached), constructs permalinks when a team is known, and collapses
// message whitespace for non-JSON output.
func renderMessages(ctx context.Context, app *appContext, title string, posts []*model.Post, permalinkTeam string, full bool) output.Result {
	usernames := resolveUsernames(ctx, app, posts)
	channelNames := map[string]string{}
	clean := app.outputMode != "json" && !full
	server := serverBase(app)

	res := output.Result{Title: title, Columns: messageColumns}
	for _, p := range posts {
		if p == nil {
			continue
		}
		user := usernames[p.UserId]
		if user == "" {
			user = p.UserId
		}
		msg := p.Message
		if clean {
			msg = preview(msg, maxMessagePreview)
		}
		row := output.Row{
			"time":    time.UnixMilli(p.CreateAt).Format(time.RFC3339),
			"channel": channelLabel(ctx, app, p.ChannelId, channelNames),
			"user":    user,
			"files":   fileSummary(p),
			"root_id": p.RootId,
			"post_id": p.Id,
			"message": msg,
		}
		if server != "" && permalinkTeam != "" {
			row["permalink"] = server + "/" + permalinkTeam + "/pl/" + p.Id
		}
		res.Rows = append(res.Rows, row)
	}
	return res
}

// resolveUsernames maps the unique author IDs of posts to usernames via a single
// batched lookup. On error it returns an empty map (callers fall back to IDs).
func resolveUsernames(ctx context.Context, app *appContext, posts []*model.Post) map[string]string {
	seen := map[string]bool{}
	var ids []string
	for _, p := range posts {
		if p == nil || p.UserId == "" || seen[p.UserId] {
			continue
		}
		seen[p.UserId] = true
		ids = append(ids, p.UserId)
	}
	out := map[string]string{}
	users, err := app.api.UsersByIDs(ctx, ids)
	if err != nil {
		return out
	}
	for _, u := range users {
		out[u.Id] = u.Username
	}
	return out
}

// channelLabel returns a human name for a channel ID, caching lookups. It falls
// back to the raw ID when the channel cannot be resolved.
func channelLabel(ctx context.Context, app *appContext, id string, cache map[string]string) string {
	if id == "" {
		return ""
	}
	if v, ok := cache[id]; ok {
		return v
	}
	label := id
	if ch, err := app.api.Channel(ctx, id); err == nil && ch != nil {
		switch {
		case ch.DisplayName != "":
			label = ch.DisplayName
		case ch.Name != "":
			label = ch.Name
		}
	}
	cache[id] = label
	return label
}

// preview collapses all runs of whitespace (including newlines and tabs) into
// single spaces and truncates to maxLen runes with an ellipsis.
func preview(s string, maxLen int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) > maxLen {
		return string(r[:maxLen]) + "…"
	}
	return s
}

// serverBase returns the server URL without a trailing slash, for permalinks.
func serverBase(app *appContext) string {
	return strings.TrimRight(app.api.ServerURL(), "/")
}

// fileSummary describes a post's attachments: the count plus filenames when the
// post carries file metadata, the count alone otherwise, or "" when there are
// none.
func fileSummary(p *model.Post) string {
	if p == nil {
		return ""
	}
	if p.Metadata != nil && len(p.Metadata.Files) > 0 {
		names := make([]string, 0, len(p.Metadata.Files))
		for _, fi := range p.Metadata.Files {
			names = append(names, fi.Name)
		}
		return fmt.Sprintf("%d: %s", len(names), strings.Join(names, ", "))
	}
	if n := len(p.FileIds); n > 0 {
		return strconv.Itoa(n)
	}
	return ""
}
