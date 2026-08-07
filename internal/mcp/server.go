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

	return &Server{
		api:       api,
		srv:       srv,
		userID:    me.Id,
		username:  me.Username,
		serverURL: serverURL,
		team:      team,
		tier:      tier,
	}, nil
}

// Run starts the MCP server on stdio, blocking until the context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	fmt.Fprintln(os.Stderr, "mmrun MCP server listening on stdio. Press Ctrl+C to stop.")
	return server.ServeStdio(s.srv)
}

// Ensure we import mcp package for future tool registration.
var _ = mcp.CallToolRequest{}
