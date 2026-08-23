package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestMentions_TeamScoped(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {
			Id: "p1", Message: "hey @alice", UserId: "u2",
			ChannelId: "c1", CreateAt: 1000,
		}},
	}
	fake := &client.FakeAPI{
		Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
		Posts_:    pl,
		Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen},
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", username: "alice", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runMentions(app, "eng", "", 30, false, "", "", "", false, true, false, &buf); err != nil {
		t.Fatalf("runMentions: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "hey @alice") {
		t.Errorf("mentions output missing post content:\n%s", out)
	}
}
