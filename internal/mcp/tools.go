package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/isdmx/mmrun/internal/output"
)

// --- helpers ----------------------------------------------------------------

func optionalString(args map[string]any, key, def string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func optionalInt(args map[string]any, key string, def int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case string:
			if i, err := strconv.Atoi(n); err == nil {
				return i
			}
		}
	}
	return def
}

// getArgs extracts a map[string]any from req.Params.Arguments.
func getArgs(req mcp.CallToolRequest) map[string]any {
	if m, ok := req.Params.Arguments.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

// messageRow builds a standard Row for a post.
func messageRow(p *model.Post, channelName, serverURL string) output.Row {
	return output.Row{
		"time":      time.UnixMilli(p.CreateAt).UTC().Format(time.RFC3339),
		"user":      p.UserId,
		"message":   p.Message,
		"post_id":   p.Id,
		"channel":   channelName,
		"root_id":   p.RootId,
		"permalink": fmt.Sprintf("%s/_redirect/pl/%s", serverURL, p.Id),
	}
}

// renderResult renders a Result via AIRenderer and returns a CallToolResult.
func renderResult(res output.Result) *mcp.CallToolResult {
	var buf bytes.Buffer
	if err := output.AIRenderer.Render(&buf, res); err != nil {
		return mcp.NewToolResultError(err.Error())
	}
	return mcp.NewToolResultText(buf.String())
}

// parseSince parses a duration string (e.g. "24h") or RFC3339 timestamp and
// returns Unix milliseconds.
func parseSince(v string) (int64, error) {
	if v == "" {
		return 0, nil
	}
	if d, err := time.ParseDuration(v); err == nil {
		return time.Now().Add(-d).UnixMilli(), nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return 0, fmt.Errorf("invalid since value %q: must be a duration (e.g. 24h) or RFC3339 timestamp", v)
	}
	return t.UnixMilli(), nil
}

// resolveTeam searches the user's teams for one whose Name or DisplayName
// matches name.
func (s *Server) resolveTeam(ctx context.Context, name string) (string, error) {
	teams, err := s.api.TeamsForUser(ctx, s.userID)
	if err != nil {
		return "", err
	}
	for _, t := range teams {
		if t.Name == name || t.DisplayName == name {
			return t.Id, nil
		}
	}
	return "", fmt.Errorf("team %q not found", name)
}

// friendlyErr wraps err with an HTTP status code prefix when available.
func friendlyErr(prefix string, err error) string {
	code := client.StatusCode(err)
	if code > 0 {
		return fmt.Sprintf("%s: HTTP %d: %v", prefix, code, err)
	}
	return fmt.Sprintf("%s: %v", prefix, err)
}

// --- read-tier handlers -----------------------------------------------------

// listThreads returns the user's followed threads with unread counts.
func (s *Server) listThreads(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	limit := optionalInt(args, "limit", 60)
	unreadOnly := optionalString(args, "unread", "false") != "false"

	threads, err := s.api.UserThreads(ctx, s.userID, "", unreadOnly, limit)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("listing threads", err)), nil
	}
	if threads == nil || len(threads.Threads) == 0 {
		return mcp.NewToolResultText("No threads."), nil
	}

	res := output.Result{
		Title:   "Threads",
		Columns: []string{"thread_id", "reply_count", "last_reply", "unread_replies", "unread_mentions"},
	}
	for _, t := range threads.Threads {
		res.Rows = append(res.Rows, output.Row{
			"thread_id":       t.PostId,
			"reply_count":     strconv.FormatInt(t.ReplyCount, 10),
			"last_reply":      time.UnixMilli(t.LastReplyAt).UTC().Format(time.RFC3339),
			"unread_replies":  strconv.FormatInt(t.UnreadReplies, 10),
			"unread_mentions": strconv.FormatInt(t.UnreadMentions, 10),
		})
	}
	return renderResult(res), nil
}

// readChannel reads messages from a channel, optionally since a timestamp.
func (s *Server) readChannel(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	chRef := optionalString(args, "channel", "")
	if chRef == "" {
		return mcp.NewToolResultError("channel is required"), nil
	}

	ch, err := s.api.ResolveChannel(ctx, chRef, s.team, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolving channel", err)), nil
	}

	sinceStr := optionalString(args, "since", "")
	since, err := parseSince(sinceStr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var posts *model.PostList
	if since > 0 {
		posts, err = s.api.PostsSince(ctx, ch.Id, since)
	} else {
		posts, err = s.api.PostsForChannel(ctx, ch.Id, 60)
	}
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("reading channel", err)), nil
	}

	sorted := client.SortPosts(posts)
	rows := make([]output.Row, 0, len(sorted))
	for _, p := range sorted {
		rows = append(rows, messageRow(p, ch.DisplayName, s.serverURL))
	}

	if len(rows) == 0 {
		return mcp.NewToolResultText("No messages in channel."), nil
	}

	return renderResult(output.Result{
		Title:   fmt.Sprintf("# %s", ch.DisplayName),
		Columns: []string{"time", "user", "message", "post_id", "channel", "root_id", "permalink"},
		Rows:    rows,
	}), nil
}

// getThread reads a full thread by root post ID.
func (s *Server) getThread(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	postID := optionalString(args, "post_id", "")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}

	thread, err := s.api.PostThread(ctx, postID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("reading thread", err)), nil
	}

	sorted := client.SortPosts(thread)
	rows := make([]output.Row, 0, len(sorted))
	for _, p := range sorted {
		rows = append(rows, messageRow(p, "", s.serverURL))
	}

	if len(rows) == 0 {
		return mcp.NewToolResultText("Thread is empty."), nil
	}

	return renderResult(output.Result{
		Title:   fmt.Sprintf("Thread %s", postID),
		Columns: []string{"time", "user", "message", "post_id", "channel", "root_id", "permalink"},
		Rows:    rows,
	}), nil
}

// searchMessages searches messages across teams.
func (s *Server) searchMessages(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	query := optionalString(args, "query", "")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	teamName := optionalString(args, "team", "")
	var teamID string
	if teamName != "" {
		var err error
		teamID, err = s.resolveTeam(ctx, teamName)
		if err != nil {
			return mcp.NewToolResultError(friendlyErr("resolving team", err)), nil
		}
	}

	posts, err := s.api.Search(ctx, teamID, query, false, 60, 0)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("searching messages", err)), nil
	}

	sorted := client.SortPosts(posts)
	rows := make([]output.Row, 0, len(sorted))
	for _, p := range sorted {
		rows = append(rows, messageRow(p, "", s.serverURL))
	}

	if len(rows) == 0 {
		return mcp.NewToolResultText("No results."), nil
	}

	return renderResult(output.Result{
		Title:   fmt.Sprintf("Search: %s", query),
		Columns: []string{"time", "user", "message", "post_id", "channel", "root_id", "permalink"},
		Rows:    rows,
	}), nil
}

// listChannels lists channels in a team, optionally filtered by search.
func (s *Server) listChannels(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	teamName := optionalString(args, "team", s.team)
	if teamName == "" {
		return mcp.NewToolResultError("team is required"), nil
	}

	teamID, err := s.resolveTeam(ctx, teamName)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolving team", err)), nil
	}

	search := optionalString(args, "search", "")
	var channels []*model.Channel
	if search != "" {
		channels, err = s.api.SearchChannels(ctx, teamID, search)
	} else {
		channels, err = s.api.ChannelsForUser(ctx, teamID, s.userID)
	}
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("listing channels", err)), nil
	}

	rows := make([]output.Row, 0, len(channels))
	for _, ch := range channels {
		rows = append(rows, output.Row{
			"id":           ch.Id,
			"name":         ch.Name,
			"display_name": ch.DisplayName,
			"type":         string(ch.Type),
			"team_id":      ch.TeamId,
		})
	}

	if len(rows) == 0 {
		return mcp.NewToolResultText("No channels found."), nil
	}

	return renderResult(output.Result{
		Title:   fmt.Sprintf("Channels in %s", teamName),
		Columns: []string{"id", "name", "display_name", "type", "team_id"},
		Rows:    rows,
	}), nil
}

// listTeams lists teams the user belongs to.
func (s *Server) listTeams(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	teams, err := s.api.TeamsForUser(ctx, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("listing teams", err)), nil
	}

	rows := make([]output.Row, 0, len(teams))
	for _, t := range teams {
		rows = append(rows, output.Row{
			"id":           t.Id,
			"name":         t.Name,
			"display_name": t.DisplayName,
		})
	}

	return renderResult(output.Result{
		Title:   "Teams",
		Columns: []string{"id", "name", "display_name"},
		Rows:    rows,
	}), nil
}

// searchUsers searches users by username or name.
func (s *Server) searchUsers(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	term := optionalString(args, "term", "")
	if term == "" {
		return mcp.NewToolResultError("term is required"), nil
	}

	teamName := optionalString(args, "team", "")
	var teamID string
	if teamName != "" {
		var err error
		teamID, err = s.resolveTeam(ctx, teamName)
		if err != nil {
			return mcp.NewToolResultError(friendlyErr("resolving team", err)), nil
		}
	}

	limit := optionalInt(args, "limit", 60)

	users, err := s.api.SearchUsers(ctx, term, teamID, limit)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("searching users", err)), nil
	}

	rows := make([]output.Row, 0, len(users))
	for _, u := range users {
		rows = append(rows, output.Row{
			"id":       u.Id,
			"username": u.Username,
			"email":    u.Email,
			"nickname": u.Nickname,
		})
	}

	if len(rows) == 0 {
		return mcp.NewToolResultText("No users found."), nil
	}

	return renderResult(output.Result{
		Title:   fmt.Sprintf("Users matching %q", term),
		Columns: []string{"id", "username", "email", "nickname"},
		Rows:    rows,
	}), nil
}

// getMe returns the authenticated user's profile.
func (s *Server) getMe(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	me, err := s.api.Me(ctx)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("getting profile", err)), nil
	}

	return renderResult(output.Result{
		Title:   me.Username,
		Columns: []string{"id", "username", "email", "nickname", "first_name", "last_name"},
		Rows: []output.Row{{
			"id":         me.Id,
			"username":   me.Username,
			"email":      me.Email,
			"nickname":   me.Nickname,
			"first_name": me.FirstName,
			"last_name":  me.LastName,
		}},
	}), nil
}

// getUser returns a user's profile by username.
func (s *Server) getUser(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	username := optionalString(args, "username", "")
	if username == "" {
		return mcp.NewToolResultError("username is required"), nil
	}

	user, err := s.api.UserByUsername(ctx, username)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("getting user", err)), nil
	}

	return renderResult(output.Result{
		Title:   user.Username,
		Columns: []string{"id", "username", "email", "nickname", "first_name", "last_name"},
		Rows: []output.Row{{
			"id":         user.Id,
			"username":   user.Username,
			"email":      user.Email,
			"nickname":   user.Nickname,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
		}},
	}), nil
}

// getUserStatus returns a user's online status.
func (s *Server) getUserStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	userID := optionalString(args, "user_id", "")
	if userID == "" {
		return mcp.NewToolResultError("user_id is required"), nil
	}

	statuses, err := s.api.UsersStatuses(ctx, []string{userID})
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("getting status", err)), nil
	}
	if len(statuses) == 0 {
		return mcp.NewToolResultError("user not found"), nil
	}

	return mcp.NewToolResultText(statuses[0].Status), nil
}

// getPinnedPosts returns pinned posts in a channel.
func (s *Server) getPinnedPosts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	chRef := optionalString(args, "channel", "")
	if chRef == "" {
		return mcp.NewToolResultError("channel is required"), nil
	}

	ch, err := s.api.ResolveChannel(ctx, chRef, s.team, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolving channel", err)), nil
	}

	pinned, err := s.api.PinnedPosts(ctx, ch.Id)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("getting pinned posts", err)), nil
	}

	sorted := client.SortPosts(pinned)
	rows := make([]output.Row, 0, len(sorted))
	for _, p := range sorted {
		rows = append(rows, messageRow(p, ch.DisplayName, s.serverURL))
	}

	if len(rows) == 0 {
		return mcp.NewToolResultText("No pinned posts."), nil
	}

	return renderResult(output.Result{
		Title:   fmt.Sprintf("Pinned in # %s", ch.DisplayName),
		Columns: []string{"time", "user", "message", "post_id", "channel", "root_id", "permalink"},
		Rows:    rows,
	}), nil
}

// getFlaggedPosts returns posts the user has flagged.
func (s *Server) getFlaggedPosts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	teamName := optionalString(args, "team", "")
	var teamID string
	if teamName != "" {
		var err error
		teamID, err = s.resolveTeam(ctx, teamName)
		if err != nil {
			return mcp.NewToolResultError(friendlyErr("resolving team", err)), nil
		}
	}

	limit := optionalInt(args, "limit", 60)

	flagged, err := s.api.FlaggedPosts(ctx, s.userID, teamID, 0, limit)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("getting flagged posts", err)), nil
	}

	sorted := client.SortPosts(flagged)
	rows := make([]output.Row, 0, len(sorted))
	for _, p := range sorted {
		rows = append(rows, messageRow(p, "", s.serverURL))
	}

	if len(rows) == 0 {
		return mcp.NewToolResultText("No flagged posts."), nil
	}

	return renderResult(output.Result{
		Title:   "Flagged Posts",
		Columns: []string{"time", "user", "message", "post_id", "channel", "root_id", "permalink"},
		Rows:    rows,
	}), nil
}

// getChannelStats returns member and pinned counts for a channel.
func (s *Server) getChannelStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	chRef := optionalString(args, "channel", "")
	if chRef == "" {
		return mcp.NewToolResultError("channel is required"), nil
	}

	ch, err := s.api.ResolveChannel(ctx, chRef, s.team, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolving channel", err)), nil
	}

	stats, err := s.api.ChannelStats(ctx, ch.Id)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("getting channel stats", err)), nil
	}

	return mcp.NewToolResultText(
		fmt.Sprintf("Channel: %s\nMembers: %d\nPinned: %d",
			ch.DisplayName, stats.MemberCount, stats.PinnedPostCount)), nil
}

// getUnread returns unread message and mention counts for a channel.
func (s *Server) getUnread(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	chRef := optionalString(args, "channel", "")
	if chRef == "" {
		return mcp.NewToolResultError("channel is required"), nil
	}

	ch, err := s.api.ResolveChannel(ctx, chRef, s.team, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolving channel", err)), nil
	}

	unread, err := s.api.ChannelUnread(ctx, ch.Id, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("getting unread counts", err)), nil
	}

	return mcp.NewToolResultText(
		fmt.Sprintf("Channel: %s\nUnread messages: %d\nMentions: %d",
			ch.DisplayName, unread.MsgCount, unread.MentionCount)), nil
}

// flagPost flags a post for follow-up.
func (s *Server) flagPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	postID := optionalString(args, "post_id", "")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}

	if err := s.api.FlagPost(ctx, postID); err != nil {
		return mcp.NewToolResultError(friendlyErr("flagging post", err)), nil
	}

	return mcp.NewToolResultText("ok"), nil
}

// unflagPost removes a flag from a post.
func (s *Server) unflagPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(req)

	postID := optionalString(args, "post_id", "")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}

	if err := s.api.UnflagPost(ctx, postID); err != nil {
		return mcp.NewToolResultError(friendlyErr("unflagging post", err)), nil
	}

	return mcp.NewToolResultText("ok"), nil
}

// --- write-tier handlers ----------------------------------------------------

// postMessage creates a new post in a channel.
func (s *Server) postMessage(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	channel, _ := req.RequireString("channel")
	message, _ := req.RequireString("message")
	if channel == "" {
		return mcp.NewToolResultError("channel is required"), nil
	}
	if message == "" {
		return mcp.NewToolResultError("message is required"), nil
	}
	ch, err := s.api.ResolveChannel(ctx, channel, s.team, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolve channel", err)), nil
	}
	post := &model.Post{ChannelId: ch.Id, Message: message}
	created, err := s.api.CreatePost(ctx, post)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("create post", err)), nil
	}
	return mcp.NewToolResultText(created.Id), nil
}

// replyToThread creates a reply post in a thread.
func (s *Server) replyToThread(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, _ := req.RequireString("post_id")
	message, _ := req.RequireString("message")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}
	if message == "" {
		return mcp.NewToolResultError("message is required"), nil
	}
	thread, err := s.api.PostThread(ctx, postID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("fetch thread", err)), nil
	}
	root, ok := thread.Posts[postID]
	if !ok || root == nil {
		return mcp.NewToolResultError("thread root post not found"), nil
	}
	actualRoot := postID
	if root.RootId != "" {
		actualRoot = root.RootId
	}
	post := &model.Post{ChannelId: root.ChannelId, Message: message, RootId: actualRoot}
	created, err := s.api.CreatePost(ctx, post)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("create reply", err)), nil
	}
	return mcp.NewToolResultText(created.Id), nil
}

// addReaction adds an emoji reaction to a post.
func (s *Server) addReaction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, _ := req.RequireString("post_id")
	emoji, _ := req.RequireString("emoji")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}
	if emoji == "" {
		return mcp.NewToolResultError("emoji is required"), nil
	}
	emoji = strings.Trim(emoji, ":")
	if err := s.api.SaveReaction(ctx, postID, s.userID, emoji); err != nil {
		return mcp.NewToolResultError(friendlyErr("add reaction", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

// removeReaction removes the user's emoji reaction from a post.
func (s *Server) removeReaction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, _ := req.RequireString("post_id")
	emoji, _ := req.RequireString("emoji")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}
	if emoji == "" {
		return mcp.NewToolResultError("emoji is required"), nil
	}
	emoji = strings.Trim(emoji, ":")
	if err := s.api.DeleteReaction(ctx, postID, s.userID, emoji); err != nil {
		return mcp.NewToolResultError(friendlyErr("remove reaction", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

// editPost patches a post's message text.
func (s *Server) editPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, _ := req.RequireString("post_id")
	message, _ := req.RequireString("message")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}
	if message == "" {
		return mcp.NewToolResultError("message is required"), nil
	}
	if _, err := s.api.PatchPost(ctx, postID, message); err != nil {
		return mcp.NewToolResultError(friendlyErr("edit post", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

// uploadFile reads a file from disk and uploads it to a channel.
func (s *Server) uploadFile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	channel, _ := req.RequireString("channel")
	filePath, _ := req.RequireString("file_path")
	ch, err := s.api.ResolveChannel(ctx, channel, s.team, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolve channel", err)), nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("read file", err)), nil
	}
	filename := filepath.Base(filePath)
	resp, err := s.api.UploadFile(ctx, data, ch.Id, filename)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("upload file", err)), nil
	}
	var ids []string
	for _, fi := range resp.FileInfos {
		ids = append(ids, fi.Id)
	}
	return mcp.NewToolResultText(strings.Join(ids, "\n")), nil
}

// markChannelRead marks a channel as read for the current user.
func (s *Server) markChannelRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	channel, _ := req.RequireString("channel")
	ch, err := s.api.ResolveChannel(ctx, channel, s.team, s.userID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolve channel", err)), nil
	}
	if err := s.api.ViewChannel(ctx, s.userID, ch.Id); err != nil {
		return mcp.NewToolResultError(friendlyErr("mark channel read", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

// markThreadRead marks a thread as read for the current user.
func (s *Server) markThreadRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	threadID, _ := req.RequireString("thread_id")
	thread, err := s.api.PostThread(ctx, threadID)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("fetch thread", err)), nil
	}
	root, ok := thread.Posts[threadID]
	if !ok || root == nil {
		return mcp.NewToolResultError("thread root post not found"), nil
	}
	ch, err := s.api.Channel(ctx, root.ChannelId)
	if err != nil {
		return mcp.NewToolResultError(friendlyErr("resolve channel", err)), nil
	}
	if err := s.api.UpdateThreadRead(ctx, s.userID, ch.TeamId, threadID); err != nil {
		return mcp.NewToolResultError(friendlyErr("mark thread read", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

// --- admin-tier handlers ----------------------------------------------------

// deletePost deletes a post by ID.
func (s *Server) deletePost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, _ := req.RequireString("post_id")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}
	if err := s.api.DeletePost(ctx, postID); err != nil {
		return mcp.NewToolResultError(friendlyErr("delete post", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

// pinPost pins a post to the channel.
func (s *Server) pinPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, _ := req.RequireString("post_id")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}
	if err := s.api.PinPost(ctx, postID); err != nil {
		return mcp.NewToolResultError(friendlyErr("pin post", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

// unpinPost unpins a post from the channel.
func (s *Server) unpinPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	postID, _ := req.RequireString("post_id")
	if postID == "" {
		return mcp.NewToolResultError("post_id is required"), nil
	}
	if err := s.api.UnpinPost(ctx, postID); err != nil {
		return mcp.NewToolResultError(friendlyErr("unpin post", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}
