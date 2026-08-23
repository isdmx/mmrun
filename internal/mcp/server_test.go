package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/isdmx/mmrun/internal/client"
)

func TestParseTier(t *testing.T) {
	if got := parseTier("read"); got != TierRead {
		t.Errorf("read: got %v, want TierRead", got)
	}
	if got := parseTier("write"); got != TierWrite {
		t.Errorf("write: got %v, want TierWrite", got)
	}
	if got := parseTier("admin"); got != TierAdmin {
		t.Errorf("admin: got %v, want TierAdmin", got)
	}
	if got := parseTier("unknown"); got != TierRead {
		t.Errorf("unknown: got %v, want TierRead", got)
	}
}

func TestToolTiers_Count(t *testing.T) {
	if len(toolTiers) != 27 {
		t.Errorf("got %d tools, want 27", len(toolTiers))
	}
}

func TestToolTiers_ReadOnly(t *testing.T) {
	writeTools := []string{
		"post_message", "reply_to_thread", "add_reaction", "remove_reaction",
		"edit_post", "upload_file", "mark_channel_read", "mark_thread_read",
	}
	for _, name := range writeTools {
		if tier, ok := toolTiers[name]; !ok || tier == TierRead {
			t.Errorf("tool %q should be write-tier or higher", name)
		}
	}
}

func TestToolTiers_AdminOnly(t *testing.T) {
	adminTools := []string{"delete_post", "pin_post", "unpin_post"}
	for _, name := range adminTools {
		if tier, ok := toolTiers[name]; !ok || tier != TierAdmin {
			t.Errorf("tool %q should be admin-tier, got %v", name, tier)
		}
	}
}

func TestNew_NoAuth(t *testing.T) {
	t.Setenv("MMRUN_URL", "")
	t.Setenv("MMRUN_TOKEN", "")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	_, err := New(ServerConfig{ToolsTier: "read"})
	if err == nil {
		t.Error("expected error with no auth")
	}
}

// setupTestServer creates a Server with a FakeAPI for test isolation.
func setupTestServer(t *testing.T, tier SafetyTier) *Server {
	t.Helper()
	return &Server{
		api:       &client.FakeAPI{},
		userID:    "u1",
		username:  "testuser",
		serverURL: "https://mm.example.com",
		team:      "eng",
		tier:      tier,
	}
}

func TestReadChannel_Success(t *testing.T) {
	s := setupTestServer(t, TierRead)
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{
			"p1": {Id: "p1", Message: "hello", UserId: "u2", CreateAt: 1000, ChannelId: "c1"},
		},
	}
	s.api = &client.FakeAPI{
		Resolved_: &model.Channel{Id: "c1", Name: "general", DisplayName: "General", Type: model.ChannelTypeOpen},
		Posts_:    pl,
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"channel": "general"}}}
	result, err := s.readChannel(context.Background(), req)
	if err != nil {
		t.Fatalf("readChannel: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("expected text content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if !strings.Contains(text.Text, "hello") {
		t.Errorf("expected 'hello' in output, got: %s", text.Text)
	}
}

func TestReadChannel_Error(t *testing.T) {
	s := setupTestServer(t, TierRead)
	s.api = &client.FakeAPI{Err: fmt.Errorf("network error")}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"channel": "general"}}}
	result, err := s.readChannel(context.Background(), req)
	if err != nil {
		t.Fatalf("handler should not return error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}
}

func TestListThreads_WithUnread(t *testing.T) {
	s := setupTestServer(t, TierRead)
	s.api = &client.FakeAPI{
		Teams_: []*model.Team{{Id: "t1", Name: "eng"}},
		Threads_: &model.Threads{
			Threads: []*model.ThreadResponse{
				{PostId: "t1", UnreadReplies: 3, UnreadMentions: 1, LastReplyAt: 5000, ReplyCount: 7},
				{PostId: "t2", UnreadReplies: 0, UnreadMentions: 0, LastReplyAt: 4000, ReplyCount: 2},
			},
		},
	}

	result, _ := s.listThreads(context.Background(), mcp.CallToolRequest{})
	text := result.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "t1") {
		t.Errorf("expected thread_id=t1, got: %s", text)
	}
	if !strings.Contains(text, "unread_replies=3") {
		t.Errorf("expected unread_replies=3, got: %s", text)
	}
	if !strings.Contains(text, "unread_mentions=1") {
		t.Errorf("expected unread_mentions=1, got: %s", text)
	}
	if !strings.Contains(text, "t2") {
		t.Errorf("expected thread_id=t2, got: %s", text)
	}
}

func TestPostMessage(t *testing.T) {
	s := setupTestServer(t, TierWrite)
	s.api = &client.FakeAPI{
		Resolved_: &model.Channel{Id: "c1", Name: "general", DisplayName: "General", Type: model.ChannelTypeOpen},
		Created_:  &model.Post{Id: "newpost"},
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"channel": "general",
		"message": "deploy done",
	}}}
	result, _ := s.postMessage(context.Background(), req)
	text := result.Content[0].(mcp.TextContent).Text
	if text != "newpost" {
		t.Errorf("expected 'newpost', got %q", text)
	}
}

func TestReplyToThread(t *testing.T) {
	s := setupTestServer(t, TierWrite)
	s.api = &client.FakeAPI{
		Post_:    &model.Post{Id: "p1", ChannelId: "c1", RootId: ""},
		Created_: &model.Post{Id: "reply_id"},
	}

	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"post_id": "p1",
		"message": "got it",
	}}}
	result, _ := s.replyToThread(context.Background(), req)
	text := result.Content[0].(mcp.TextContent).Text
	if text != "reply_id" {
		t.Errorf("expected 'reply_id', got %q", text)
	}
}

func TestMarkThreadRead_UsesGetPost(t *testing.T) {
	s := setupTestServer(t, TierWrite)
	f := &client.FakeAPI{
		Post_:     &model.Post{Id: "t1", ChannelId: "c1", RootId: ""},
		Resolved_: &model.Channel{Id: "c1", TeamId: "team1"},
	}
	s.api = f
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"thread_id": "t1"}}}
	result, _ := s.markThreadRead(context.Background(), req)
	if result.IsError {
		t.Fatalf("unexpected error: %v", result)
	}
	if f.ReadThread_ != "t1" {
		t.Errorf("expected ReadThread_=%q, got %q", "t1", f.ReadThread_)
	}
}

func TestGetThread_CapsLimit(t *testing.T) {
	s := setupTestServer(t, TierRead)
	f := &client.FakeAPI{
		Thread_: &model.PostList{Order: []string{"p1"}, Posts: map[string]*model.Post{
			"p1": {Id: "p1", Message: "hi", UserId: "u2", CreateAt: 1000},
		}},
	}
	s.api = f
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{"post_id": "p1", "limit": 50}}}
	result, _ := s.getThread(context.Background(), req)
	if result.IsError {
		t.Fatalf("unexpected error: %v", result)
	}
	if f.PostThreadPerPage_ != 50 {
		t.Errorf("expected perPage 50, got %d", f.PostThreadPerPage_)
	}
}

func TestListThreads_UsesCachedTeam(t *testing.T) {
	s := setupTestServer(t, TierRead)
	s.teamID = "t1"
	f := &client.FakeAPI{
		Threads_: &model.Threads{Threads: []*model.ThreadResponse{{PostId: "t1"}}},
	}
	s.api = f
	_, _ = s.listThreads(context.Background(), mcp.CallToolRequest{})
	if f.TeamsForUserCalls_ != 0 {
		t.Errorf("expected 0 TeamsForUser calls with cached team, got %d", f.TeamsForUserCalls_)
	}
}
