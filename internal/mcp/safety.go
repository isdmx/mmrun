// Package mcp provides the MCP server that bridges Mattermost to LLM clients.
package mcp

// SafetyTier is the privilege level for a tool.
type SafetyTier int

// Safety tiers for tool access control.
const (
	TierRead SafetyTier = iota
	TierWrite
	TierAdmin
)

// parseTier maps a config string to a SafetyTier.
func parseTier(s string) SafetyTier {
	switch s {
	case "write":
		return TierWrite
	case "admin":
		return TierAdmin
	default:
		return TierRead
	}
}

// toolTiers maps every tool by name to its required SafetyTier.
var toolTiers = map[string]SafetyTier{
	// read tier (16 tools)
	"list_threads":      TierRead,
	"read_channel":      TierRead,
	"get_thread":        TierRead,
	"search_messages":   TierRead,
	"list_channels":     TierRead,
	"list_teams":        TierRead,
	"search_users":      TierRead,
	"get_me":            TierRead,
	"get_user":          TierRead,
	"get_user_status":   TierRead,
	"get_pinned_posts":  TierRead,
	"get_flagged_posts": TierRead,
	"get_channel_stats": TierRead,
	"get_unread":        TierRead,
	"flag_post":         TierRead,
	"unflag_post":       TierRead,

	// write tier (8 tools)
	"post_message":      TierWrite,
	"reply_to_thread":   TierWrite,
	"add_reaction":      TierWrite,
	"remove_reaction":   TierWrite,
	"edit_post":         TierWrite,
	"upload_file":       TierWrite,
	"mark_channel_read": TierWrite,
	"mark_thread_read":  TierWrite,

	// admin tier (3 tools)
	"delete_post": TierAdmin,
	"pin_post":    TierAdmin,
	"unpin_post":  TierAdmin,
}
