package mcp

import "testing"

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
