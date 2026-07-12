package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestChannelList(t *testing.T) {
	app := &appContext{
		api: &fakeAPI{
			teams:    []*model.Team{{Id: "t1", Name: "eng"}},
			channels: []*model.Channel{{Id: "c1", Name: "general", DisplayName: "General"}},
		},
		outputMode: "ai",
		userID:     "u1",
	}
	var buf bytes.Buffer
	if err := runChannelList(app, "eng", &buf); err != nil {
		t.Fatalf("runChannelList: %v", err)
	}
	if !strings.Contains(buf.String(), "general") {
		t.Errorf("missing channel:\n%s", buf.String())
	}
}
