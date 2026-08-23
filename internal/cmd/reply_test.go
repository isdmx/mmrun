package cmd

import (
	"bytes"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestReply(t *testing.T) {
	fake := &client.FakeAPI{
		Thread_: &model.PostList{
			Posts: map[string]*model.Post{"p1": {Id: "p1", ChannelId: "c1", RootId: ""}},
		},
		Created_: &model.Post{Id: "r1"},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runReply(app, "p1", "my reply", postOpts{}, &buf); err != nil {
		t.Fatalf("reply: %v", err)
	}
	if fake.LastPost_ == nil || fake.LastPost_.RootId != "p1" {
		t.Errorf("reply should be threaded under p1: %+v", fake.LastPost_)
	}
	if fake.LastPost_ == nil || fake.LastPost_.ChannelId != "c1" {
		t.Errorf("reply should be in same channel c1: %+v", fake.LastPost_)
	}
}
