package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestTeamList(t *testing.T) {
	app := &appContext{
		api:        &fakeAPI{teams: []*model.Team{{Id: "t1", Name: "eng", DisplayName: "Engineering"}}},
		outputMode: "ai",
		userID:     "u1",
	}
	var buf bytes.Buffer
	if err := runTeamList(app, &buf); err != nil {
		t.Fatalf("runTeamList: %v", err)
	}
	if !strings.Contains(buf.String(), "eng") {
		t.Errorf("missing team:\n%s", buf.String())
	}
}
