package cmd

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestReply(t *testing.T) {
	fake := &fakeAPI{
		thread: &model.PostList{
			Posts: map[string]*model.Post{"p1": {Id: "p1", ChannelId: "c1", RootId: ""}},
		},
		created: &model.Post{Id: "r1"},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runReply(app, "p1", "my reply", postOpts{}, &buf); err != nil {
		t.Fatalf("reply: %v", err)
	}
	if fake.lastPost == nil || fake.lastPost.RootId != "p1" {
		t.Errorf("reply should be threaded under p1: %+v", fake.lastPost)
	}
	if fake.lastPost == nil || fake.lastPost.ChannelId != "c1" {
		t.Errorf("reply should be in same channel c1: %+v", fake.lastPost)
	}
}
