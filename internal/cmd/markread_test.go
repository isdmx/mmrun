package cmd

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestMarkRead_Channel(t *testing.T) {
	fake := &fakeAPI{resolved: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runMarkRead(app, "c1", "channel", &buf); err != nil {
		t.Fatalf("markRead: %v", err)
	}
	if fake.viewedChannel != "c1" {
		t.Error("expected ViewChannel to be called")
	}
}

func TestMarkRead_Thread(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runMarkRead(app, "p1", "thread", &buf); err != nil {
		t.Fatalf("markRead thread: %v", err)
	}
	if fake.readThread != "p1" {
		t.Error("expected UpdateThreadRead to be called")
	}
}
