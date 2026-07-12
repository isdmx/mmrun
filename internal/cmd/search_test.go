package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestSearch_RendersHits(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "deploy failed", UserId: "u2", CreateAt: 5000}},
	}
	app := &appContext{
		api:        &fakeAPI{teams: []*model.Team{{Id: "t1", Name: "eng"}}, posts: pl},
		outputMode: "ai",
		userID:     "u1",
	}
	var buf bytes.Buffer
	if err := runSearch(app, "deploy", "eng", &buf); err != nil {
		t.Fatalf("runSearch: %v", err)
	}
	if !strings.Contains(buf.String(), "deploy failed") {
		t.Errorf("missing hit:\n%s", buf.String())
	}
}
