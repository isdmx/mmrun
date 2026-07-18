package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestPinned(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "pinned msg", UserId: "u2", ChannelId: "c1", CreateAt: 1000}},
	}
	fake := &fakeAPI{resolved: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, pinnedPosts: pl, users: []*model.User{{Id: "u2", Username: "bob"}}}
	app := &appContext{api: fake, outputMode: "ai", previewLen: 140}
	var buf bytes.Buffer
	if err := runPinned(app, "general", "", false, "", "", "", &buf); err != nil {
		t.Fatalf("pinned: %v", err)
	}
	if !strings.Contains(buf.String(), "pinned msg") {
		t.Error("should show pinned post")
	}
}
