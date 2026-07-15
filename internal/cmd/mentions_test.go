package cmd

import (
	"bytes"
	"strings"
	"testing"

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
	fake := &fakeAPI{
		teams:    []*model.Team{{Id: "t1", Name: "eng"}},
		posts:    pl,
		resolved: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen},
		users:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", username: "alice", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runMentions(app, "eng", "", 30, false, "", &buf); err != nil {
		t.Fatalf("runMentions: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "hey @alice") {
		t.Errorf("mentions output missing post content:\n%s", out)
	}
}
