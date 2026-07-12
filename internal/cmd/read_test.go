package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestRead_RendersMessagesInOrder(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p2", "p1"},
		Posts: map[string]*model.Post{
			"p1": {Id: "p1", Message: "first", CreateAt: 1000},
			"p2": {Id: "p2", Message: "second", CreateAt: 2000},
		},
	}
	app := &appContext{
		api:        &fakeAPI{resolved: &model.Channel{Id: "c1"}, posts: pl},
		outputMode: "ai",
	}
	var buf bytes.Buffer
	if err := runRead(app, "eng/general", readOpts{limit: 50}, &buf); err != nil {
		t.Fatalf("runRead: %v", err)
	}
	out := buf.String()
	if strings.Index(out, "first") > strings.Index(out, "second") {
		t.Errorf("messages not in chronological order:\n%s", out)
	}
}
