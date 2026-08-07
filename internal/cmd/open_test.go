package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestResolveOpenURL_Post(t *testing.T) {
	fake := &client.FakeAPI{
		Thread_:   &model.PostList{Posts: map[string]*model.Post{"p1": {Id: "p1", ChannelId: "c1"}}},
		Resolved_: &model.Channel{Id: "c1", TeamId: "t1", Name: "general"},
		Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
	}
	app := &appContext{api: fake, userID: "u1"}
	url, err := resolveOpenURL(context.Background(), app, "p1")
	if err != nil {
		t.Fatalf("resolveOpenURL: %v", err)
	}
	if !strings.Contains(url, "/eng/pl/p1") {
		t.Errorf("url = %q, want .../eng/pl/p1", url)
	}
}

func TestResolveOpenURL_Channel(t *testing.T) {
	fake := &client.FakeAPI{
		Resolved_: &model.Channel{Id: "c1", TeamId: "t1", Name: "general", Type: model.ChannelTypeOpen},
		Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
	}
	app := &appContext{api: fake, userID: "u1"}
	url, err := resolveOpenURL(context.Background(), app, "c1")
	if err != nil {
		t.Fatalf("resolveOpenURL: %v", err)
	}
	if !strings.Contains(url, "/eng/channels/general") {
		t.Errorf("url = %q, want .../eng/channels/general", url)
	}
}
