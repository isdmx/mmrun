package cmd

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/isdmx/mmrun/internal/output"
)

var messageColumns = []string{"time", "channel", "user", "files", "reactions", "root_id", "post_id", "permalink", "message"}

// renderMessages builds a message Result from posts in the given order. It
// resolves user IDs to usernames (one batched call), channel IDs to readable
// names (cached), constructs permalinks when a team is known, and collapses
// message whitespace for non-JSON output.
func renderMessages(ctx context.Context, app *appContext, title string, posts []*model.Post, permalinkTeam string, full bool, columns []string, hideChannel bool) output.Result {
	usernames := resolveUsernames(ctx, app, posts)
	statuses := resolveStatuses(ctx, app, posts)
	reactions := resolveReactions(ctx, app, posts)
	channelNames := map[string]string{}
	clean := app.outputMode != "json" && !full
	server := serverBase(app)

	res := output.Result{Title: title, Columns: columns}
	for _, p := range posts {
		if p == nil {
			continue
		}
		user := usernames[p.UserId]
		if user == "" {
			user = p.UserId
		} else {
			user = "@" + user
		}
		user = statuses[p.UserId] + user
		msg := p.Message
		if clean {
			msg = preview(msg, app.previewLen)
		}
		row := output.Row{
			"time":      time.UnixMilli(p.CreateAt).Format(time.RFC3339),
			"user":      user,
			"files":     fileSummary(p),
			"reactions": reactions[p.Id],
			"root_id":   p.RootId,
			"post_id":   p.Id,
			"message":   msg,
		}
		if !hideChannel {
			row["channel"] = channelLabel(ctx, app, p.ChannelId, channelNames)
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

// channelLabel returns a human name for a channel ID, caching lookups. Direct
// messages resolve to the peer's @username (or "you" for a self-DM); other
// channels use their display name or name; unresolved IDs fall back to the raw
// ID.
func channelLabel(ctx context.Context, app *appContext, id string, cache map[string]string) string {
	if id == "" {
		return ""
	}
	if v, ok := cache[id]; ok {
		return v
	}
	label := id
	if ch, err := app.api.Channel(ctx, id); err == nil && ch != nil {
		if ch.Type == model.ChannelTypeDirect {
			label = directLabel(ctx, app, ch)
		} else {
			switch {
			case ch.DisplayName != "":
				label = ch.DisplayName
			case ch.Name != "":
				label = ch.Name
			}
		}
	}
	cache[id] = label
	return label
}

// directLabel resolves a direct-message channel to "@peer" or "you" (self-DM).
func directLabel(ctx context.Context, app *appContext, ch *model.Channel) string {
	var peer string
	for _, part := range strings.Split(ch.Name, "__") {
		if part != app.userID {
			peer = part
		}
	}
	if peer == "" { // self-DM (u1__u1)
		return "you"
	}
	if users, err := app.api.UsersByIDs(ctx, []string{peer}); err == nil {
		for _, u := range users {
			if u.Id == peer {
				return "@" + u.Username
			}
		}
	}
	return ch.Name
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

// resolveReactions fetches reactions for each post (bounded concurrency, 8 at a
// time) and returns a postId→display summary map, e.g. ":thumbsup: 2 :rocket: 1".
// Note: reactions may not be returned for DM channel posts by some Mattermost
// servers — this is a server-side limitation, not a client bug.
func resolveReactions(ctx context.Context, app *appContext, posts []*model.Post) map[string]string {
	out := map[string]string{}
	launched := 0
	for _, p := range posts {
		if p != nil {
			launched++
		}
	}
	if launched == 0 {
		return out
	}
	type result struct{ id, display string }
	results := make(chan result, launched)
	sem := make(chan struct{}, 8)

	for _, p := range posts {
		if p == nil {
			continue
		}
		sem <- struct{}{}
		go func(postID string) {
			defer func() { <-sem }()
			rr, err := app.api.ReactionsForPost(ctx, postID)
			if err != nil {
				results <- result{id: postID}
				return
			}
			counts := map[string]int{}
			for _, r := range rr {
				counts[r.EmojiName]++
			}
			var parts []string
			for _, name := range sortedKeys(counts) {
				parts = append(parts, fmt.Sprintf(":%s: %d", name, counts[name]))
			}
			results <- result{id: postID, display: strings.Join(parts, " ")}
		}(p.Id)
	}

	for i := 0; i < launched; i++ {
		r := <-results
		if r.display != "" {
			out[r.id] = r.display
		}
	}
	return out
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func resolveStatuses(ctx context.Context, app *appContext, posts []*model.Post) map[string]string {
	seen := map[string]bool{}
	var ids []string
	for _, p := range posts {
		if p == nil || seen[p.UserId] {
			continue
		}
		seen[p.UserId] = true
		ids = append(ids, p.UserId)
	}
	ss, err := app.api.UsersStatuses(ctx, ids)
	if err != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, s := range ss {
		switch s.Status {
		case "online":
			out[s.UserId] = "🟢"
		case "away":
			out[s.UserId] = "🌙"
		case "dnd":
			out[s.UserId] = "⛔"
		default:
			out[s.UserId] = "⛔"
		}
	}
	return out
}

func filterRoots(posts []*model.Post) []*model.Post {
	out := make([]*model.Post, 0, len(posts))
	for _, p := range posts {
		if p != nil && p.RootId == "" {
			out = append(out, p)
		}
	}
	return out
}
