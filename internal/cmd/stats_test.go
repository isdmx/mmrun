package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestStats(t *testing.T) {
	fake := &client.FakeAPI{Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, ChannelStats_: &model.ChannelStats{MemberCount: 47, PinnedPostCount: 3}}
	app := &appContext{api: fake, outputMode: "ai"}
	var buf bytes.Buffer
	if err := runStats(app, "general", &buf); err != nil {
		t.Fatalf("stats: %v", err)
	}
	if !strings.Contains(buf.String(), "47") {
		t.Error("should show member count")
	}
}
