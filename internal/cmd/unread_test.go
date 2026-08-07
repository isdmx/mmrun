package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestUnread(t *testing.T) {
	ch1 := &model.Channel{Id: "c1", Name: "general", DisplayName: "General", Type: model.ChannelTypeOpen}
	ch2 := &model.Channel{Id: "c2", Name: "random", DisplayName: "Random", Type: model.ChannelTypeOpen}
	app := &appContext{
		api: &client.FakeAPI{
			Teams_:         []*model.Team{{Id: "t1", Name: "eng"}},
			Channels_:      []*model.Channel{ch1, ch2},
			ChannelUnread_: &model.ChannelUnread{MsgCount: 5, MentionCount: 2},
		},
		outputMode: "ai",
		userID:     "u1",
	}
	var buf bytes.Buffer
	if err := runUnread(app, "eng", &buf); err != nil {
		t.Fatalf("unread: %v", err)
	}
	if !strings.Contains(buf.String(), "General") {
		t.Errorf("missing channel General:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "Random") {
		t.Errorf("missing channel Random:\n%s", buf.String())
	}
}
