package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/isdmx/mmrun/internal/config"
	"github.com/isdmx/mmrun/internal/session"
	"github.com/isdmx/mmrun/internal/version"
)

// ServerConfig holds startup configuration for the MCP server.
type ServerConfig struct {
	ToolsTier string
	SkillsDir string
	Team      string
}

// Server is the MCP stdio server that bridges Mattermost tools to MCP clients.
type Server struct {
	api       client.API
	srv       *server.MCPServer
	userID    string
	username  string
	serverURL string
	team      string
	tier      SafetyTier
}

// New builds a Server, loading auth from env vars or session and registering tools.
func New(cfg ServerConfig) (*Server, error) {
	tier := parseTier(cfg.ToolsTier)

	// 1. Load auth — env vars preferred, then session file.
	var serverURL, token string
	if url := os.Getenv("MMRUN_URL"); url != "" {
		if tok := os.Getenv("MMRUN_TOKEN"); tok != "" {
			serverURL, token = url, tok
		}
	}
	if token == "" {
		s, err := session.Load()
		if err != nil {
			return nil, fmt.Errorf("no active session: %w. Run 'mmrun auth login' or set MMRUN_URL and MMRUN_TOKEN", err)
		}
		serverURL, token = s.ServerURL, s.Token
	}

	// 2. Build API client.
	api := client.NewWithToken(serverURL, token, nil)

	// 3. Resolve user identity.
	me, err := api.Me(context.Background())
	if err != nil {
		return nil, fmt.Errorf("auth validation failed: %w", err)
	}

	// 4. Default team — explicit config wins, then config file.
	team := cfg.Team
	if team == "" {
		c, _ := config.Load()
		if c != nil {
			team = c.DefaultTeam
		}
	}

	// 5. Create MCP server.
	srv := server.NewMCPServer("mmrun", version.String(),
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
	)

	// 6. Build and register tools.
	s := &Server{
		api:       api,
		srv:       srv,
		userID:    me.Id,
		username:  me.Username,
		serverURL: serverURL,
		team:      team,
		tier:      tier,
	}
	s.registerTools()
	return s, nil
}

// Run starts the MCP server on stdio, blocking until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	fmt.Fprintln(os.Stderr, "mmrun MCP server listening on stdio. Press Ctrl+C to stop.")
	return server.ServeStdio(s.srv)
}

// registerTools registers every tool whose SafetyTier is at or below the
// server's configured tier.
func (s *Server) registerTools() {
	register := func(name, desc string, handler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
		if s.tier >= toolTiers[name] {
			s.srv.AddTool(mcp.NewTool(name, mcp.WithDescription(desc)), handler)
		}
	}
	register("get_inbox", "Get unread messages and followed threads across all teams", s.getInbox)
	register("read_channel", "Read messages from a channel", s.readChannel)
	register("get_thread", "Read a full thread by root post ID", s.getThread)
	register("search_messages", "Search messages across teams", s.searchMessages)
	register("list_channels", "List channels in a team", s.listChannels)
	register("list_teams", "List teams the user belongs to", s.listTeams)
	register("search_users", "Search users by username or name", s.searchUsers)
	register("get_me", "Get the authenticated user's profile", s.getMe)
	register("get_user", "Get a user's profile by username", s.getUser)
	register("get_user_status", "Get a user's online status", s.getUserStatus)
	register("get_pinned_posts", "Get pinned posts in a channel", s.getPinnedPosts)
	register("get_flagged_posts", "Get posts the user has flagged", s.getFlaggedPosts)
	register("get_channel_stats", "Get member and pinned count for a channel", s.getChannelStats)
	register("get_unread", "Get unread message and mention count for a channel", s.getUnread)
	register("flag_post", "Flag a post for follow-up", s.flagPost)
	register("unflag_post", "Remove flag from a post", s.unflagPost)
	register("post_message", "Post a new message to a channel", s.postMessage)
	register("reply_to_thread", "Reply to a post in its thread", s.replyToThread)
	register("add_reaction", "Add an emoji reaction to a post", s.addReaction)
	register("remove_reaction", "Remove your emoji reaction from a post", s.removeReaction)
	register("edit_post", "Edit a post's message text", s.editPost)
	register("upload_file", "Upload a file to a channel", s.uploadFile)
	register("mark_channel_read", "Mark a channel as read", s.markChannelRead)
	register("mark_thread_read", "Mark a thread as read", s.markThreadRead)
	register("delete_post", "Delete a post", s.deletePost)
	register("pin_post", "Pin a post to the channel", s.pinPost)
	register("unpin_post", "Unpin a post from the channel", s.unpinPost)
}
