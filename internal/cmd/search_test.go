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
	if err := runSearch(app, "test", "eng", false, "", "", "", "", 20, 0, "", "", false, true, &buf); err != nil {
		t.Fatalf("runSearch with limit: %v", err)
	}
}

func TestSearch_SinceModifier(t *testing.T) {
	fake := &fakeAPI{posts: &model.PostList{}, teams: []*model.Team{{Id: "t1", Name: "eng"}}}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runSearch(app, "test", "eng", false, "", "", "", "", 0, 0, "2026-07-01", "", false, true, &buf); err != nil {
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
	if err := runSearch(app, "deploy", "eng", false, "", "", "", "", 0, 0, "", "", false, true, &buf); err != nil {
		t.Fatalf("runSearch: %v", err)
	}
	if !strings.Contains(buf.String(), "deploy failed") {
		t.Errorf("missing hit:\n%s", buf.String())
	}
}
