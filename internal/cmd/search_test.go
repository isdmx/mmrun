package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestSearch_Limit(t *testing.T) {
	fake := &fakeAPI{posts: &model.PostList{}, teams: []*model.Team{{Id: "t1", Name: "eng"}}}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runSearch(app, "test", "eng", false, "", "", "", "", 20, 0, "", "", "", "", "", false, true, &buf); err != nil {
		t.Fatalf("runSearch with limit: %v", err)
	}
}

func TestSearch_SinceModifier(t *testing.T) {
	fake := &fakeAPI{posts: &model.PostList{}, teams: []*model.Team{{Id: "t1", Name: "eng"}}}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runSearch(app, "test", "eng", false, "", "", "", "", 0, 0, "2026-07-01", "", "", "", "", false, true, &buf); err != nil {
		t.Fatalf("runSearch with since: %v", err)
	}
}

func TestSearch_RendersHits(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "deploy failed", UserId: "u2", CreateAt: 5000}},
	}
	app := &appContext{
		api:        &fakeAPI{teams: []*model.Team{{Id: "t1", Name: "eng"}}, posts: pl},
		outputMode: "ai",
		userID:     "u1",
		previewLen: 140,
	}
	var buf bytes.Buffer
	if err := runSearch(app, "deploy", "eng", false, "", "", "", "", 0, 0, "", "", "", "", "", false, true, &buf); err != nil {
		t.Fatalf("runSearch: %v", err)
	}
	if !strings.Contains(buf.String(), "deploy failed") {
		t.Errorf("missing hit:\n%s", buf.String())
	}
}

func TestSearch_FromUser(t *testing.T) {
	fake := &fakeAPI{posts: &model.PostList{}, teams: []*model.Team{{Id: "t1", Name: "eng"}}}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runSearch(app, "test", "eng", false, "", "", "", "", 0, 0, "", "", "bob", "", "", false, true, &buf); err != nil {
		t.Fatalf("runSearch with from: %v", err)
	}
	if !strings.Contains(fake.searchTerms, "from:bob") {
		t.Errorf("query should contain from:bob, got %q", fake.searchTerms)
	}
}

func TestSearch_AllTeams(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1", "p2"},
		Posts: map[string]*model.Post{
			"p1": {Id: "p1", Message: "team one msg", UserId: "u1", CreateAt: 5000},
			"p2": {Id: "p2", Message: "team two msg", UserId: "u1", CreateAt: 6000},
		},
	}
	fake := &fakeAPI{
		teams: []*model.Team{{Id: "t1", Name: "eng"}, {Id: "t2", Name: "ops"}},
		posts: pl,
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runSearch(app, "test", "", false, "", "", "", "", 0, 0, "", "", "", "", "", false, true, &buf); err != nil {
		t.Fatalf("runSearch all teams: %v", err)
	}
	if !strings.Contains(buf.String(), "team one msg") {
		t.Errorf("missing team one msg:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "team two msg") {
		t.Errorf("missing team two msg:\n%s", buf.String())
	}
	if fake.searchCalls != 2 {
		t.Errorf("expected 2 search calls, got %d", fake.searchCalls)
	}
}
