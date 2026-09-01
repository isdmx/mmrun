package cmd

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestDMList_FiltersToDMAndGMChannels(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1", "p2", "p3"},
		Posts: map[string]*model.Post{
			"p1": {Id: "p1", Message: "dm hello", UserId: "u2", ChannelId: "dm1", CreateAt: 2000},
			"p2": {Id: "p2", Message: "gm hello", UserId: "u3", ChannelId: "gm1", CreateAt: 3000},
			"p3": {Id: "p3", Message: "channel noise", UserId: "u4", ChannelId: "c1", CreateAt: 1000},
		},
	}
	fake := &client.FakeAPI{
		Teams_: []*model.Team{{Id: "t1", Name: "eng"}},
		Channels_: []*model.Channel{
			{Id: "dm1", Name: "u1__u2", Type: model.ChannelTypeDirect},
			{Id: "gm1", Name: "group", Type: model.ChannelTypeGroup},
			{Id: "c1", Name: "general", Type: model.ChannelTypeOpen},
		},
		Posts_: pl,
		Users_: []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runDMList(app, dmOpts{since: "24h", limit: 30}, &buf); err != nil {
		t.Fatalf("runDMList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "dm hello") {
		t.Errorf("output missing DM post:\n%s", out)
	}
	if !strings.Contains(out, "gm hello") {
		t.Errorf("output missing GM post:\n%s", out)
	}
	if strings.Contains(out, "channel noise") {
		t.Errorf("output contains non-DM channel post:\n%s", out)
	}
}

func TestDMList_SinceModifier(t *testing.T) {
	fake := &client.FakeAPI{
		Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
		Channels_: []*model.Channel{{Id: "dm1", Name: "u1__u2", Type: model.ChannelTypeDirect}},
		Posts_:    &model.PostList{Order: nil, Posts: map[string]*model.Post{}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runDMList(app, dmOpts{since: "2026-09-01", limit: 10}, &buf); err != nil {
		t.Fatalf("runDMList: %v", err)
	}
	if !regexp.MustCompile(` after:\d{4}-\d{2}-\d{2}`).MatchString(fake.SearchTerms_) {
		t.Errorf("search terms missing after: modifier: %q", fake.SearchTerms_)
	}
}

func TestDMList_DefaultSince24h(t *testing.T) {
	fake := &client.FakeAPI{
		Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
		Channels_: nil,
		Posts_:    &model.PostList{Order: nil, Posts: map[string]*model.Post{}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runDMList(app, dmOpts{limit: 10}, &buf); err != nil {
		t.Fatalf("runDMList: %v", err)
	}
	if !regexp.MustCompile(` after:\d{4}-\d{2}-\d{2}`).MatchString(fake.SearchTerms_) {
		t.Errorf("default --since should produce an after: modifier: %q", fake.SearchTerms_)
	}
}

func TestDMList_MultiTeamDedupeAndSort(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1", "p2"},
		Posts: map[string]*model.Post{
			"p1": {Id: "p1", Message: "later", UserId: "u2", ChannelId: "dm1", CreateAt: 5000},
			"p2": {Id: "p2", Message: "earlier", UserId: "u2", ChannelId: "dm1", CreateAt: 4000},
		},
	}
	fake := &client.FakeAPI{
		Teams_: []*model.Team{{Id: "t1", Name: "a"}, {Id: "t2", Name: "b"}},
		Channels_: []*model.Channel{
			{Id: "dm1", Name: "u1__u2", Type: model.ChannelTypeDirect},
		},
		Posts_: pl,
		Users_: []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runDMList(app, dmOpts{limit: 30}, &buf); err != nil {
		t.Fatalf("runDMList: %v", err)
	}
	if got := fake.SearchCalls_; got != 2 {
		t.Errorf("Search calls = %d, want 2 (one per team)", got)
	}
	out := buf.String()
	if strings.Count(out, "later") != 1 || strings.Count(out, "earlier") != 1 {
		t.Errorf("posts should be deduped across teams:\n%s", out)
	}
	if strings.Index(out, "earlier") > strings.Index(out, "later") {
		t.Errorf("posts should be sorted ascending by CreateAt:\n%s", out)
	}
}

func TestDMCmd_HasListSubcommand(t *testing.T) {
	mode := "auto"
	cmd := newDMCmd(&mode)
	if cmd.Short != "List direct and group messages" {
		t.Errorf("Short = %q, want %q", cmd.Short, "List direct and group messages")
	}
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Name() == "list" {
			found = true
		}
	}
	if !found {
		t.Error("dm command missing list subcommand")
	}
}
