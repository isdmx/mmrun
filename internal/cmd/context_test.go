package cmd

import (
	"context"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestAppContext_UsesFake(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai"}
	got := app.api
	if got == nil {
		t.Fatal("api not wired")
	}
}

func TestAliases(t *testing.T) {
	fake := &fakeAPI{
		resolved: &model.Channel{Id: "c2", Name: "incidents", TeamId: "t2"},
	}
	app := &appContext{
		api:     fake,
		aliases: map[string]string{"alerts": "eng/incidents"},
	}
	ch, err := app.resolveChannel(context.Background(), "alerts", "")
	if err != nil {
		t.Fatalf("resolveChannel: %v", err)
	}
	if ch.Id != "c2" {
		t.Errorf("channel = %s, want c2", ch.Id)
	}
}
