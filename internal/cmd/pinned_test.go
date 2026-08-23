package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestPinned(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "pinned msg", UserId: "u2", ChannelId: "c1", CreateAt: 1000}},
	}
	fake := &client.FakeAPI{Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, PinnedPosts_: pl, Users_: []*model.User{{Id: "u2", Username: "bob"}}}
	app := &appContext{api: fake, outputMode: "ai", previewLen: 140}
	var buf bytes.Buffer
	if err := runPinned(app, "general", "", false, "", "", "", true, false, &buf); err != nil {
		t.Fatalf("pinned: %v", err)
	}
	if !strings.Contains(buf.String(), "pinned msg") {
		t.Error("should show pinned post")
	}
}
