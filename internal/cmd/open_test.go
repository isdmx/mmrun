package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestResolveOpenURL_Post(t *testing.T) {
	fake := &fakeAPI{
		thread:   &model.PostList{Posts: map[string]*model.Post{"p1": {Id: "p1", ChannelId: "c1"}}},
		resolved: &model.Channel{Id: "c1", TeamId: "t1", Name: "general"},
		teams:    []*model.Team{{Id: "t1", Name: "eng"}},
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
	fake := &fakeAPI{
		resolved: &model.Channel{Id: "c1", TeamId: "t1", Name: "general", Type: model.ChannelTypeOpen},
		teams:    []*model.Team{{Id: "t1", Name: "eng"}},
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
