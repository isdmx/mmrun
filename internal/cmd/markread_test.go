package cmd

import (
	"bytes"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestMarkRead_Channel(t *testing.T) {
	fake := &client.FakeAPI{Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runMarkRead(app, "c1", "channel", &buf); err != nil {
		t.Fatalf("markRead: %v", err)
	}
	if fake.ViewedChannel_ != "c1" {
		t.Error("expected ViewChannel to be called")
	}
}

func TestMarkRead_Thread(t *testing.T) {
	fake := &client.FakeAPI{
		Thread_:   &model.PostList{Posts: map[string]*model.Post{"p1": {Id: "p1", ChannelId: "c1"}}},
		Resolved_: &model.Channel{Id: "c1", TeamId: "t1"},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runMarkRead(app, "p1", "thread", &buf); err != nil {
		t.Fatalf("markRead thread: %v", err)
	}
	if fake.ReadThread_ != "p1" {
		t.Error("expected UpdateThreadRead to be called")
	}
}
