package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestPost_CreatesAndReportsID(t *testing.T) {
	app := &appContext{
		api: &fakeAPI{
			resolved: &model.Channel{Id: "c1"},
			created:  &model.Post{Id: "p1", Message: "hello"},
		},
		outputMode: "ai",
	}
	var buf bytes.Buffer
	err := runPost(app, "eng/general", "hello", postOpts{}, &buf)
	if err != nil {
		t.Fatalf("runPost: %v", err)
	}
	if !strings.Contains(buf.String(), "p1") {
		t.Errorf("expected post id in output:\n%s", buf.String())
	}
}
