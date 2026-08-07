package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestFlagged(t *testing.T) {
	pl := &model.PostList{Order: []string{"p1"}, Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "bad", UserId: "u2", CreateAt: 1000, ChannelId: "c1"}}}
	fake := &client.FakeAPI{FlaggedPosts_: pl, Users_: []*model.User{{Id: "u2", Username: "bob"}}, Teams_: []*model.Team{{Id: "t1", Name: "eng"}}, Resolved_: &model.Channel{Id: "c1", Name: "g", Type: model.ChannelTypeOpen}}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runFlagged(app, "eng", 50, "", false, "", "", "", true, false, &buf); err != nil {
		t.Fatalf("flagged: %v", err)
	}
	if !strings.Contains(buf.String(), "bad") {
		t.Error("should show flagged post")
	}
}

func TestFlag(t *testing.T) {
	fake := &client.FakeAPI{}
	app := &appContext{api: fake}
	if err := app.api.FlagPost(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if fake.PostFlagged_ != "p1" {
		t.Error("FlagPost not called")
	}
}

func TestUnflag_RequiresYes(t *testing.T) {
	fake := &client.FakeAPI{}
	app := &appContext{api: fake}
	if err := app.api.UnflagPost(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if fake.PostFlagged_ != "" {
		t.Error("UnflagPost should clear")
	}
}
