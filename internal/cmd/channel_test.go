package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestChannelList(t *testing.T) {
	app := &appContext{
		api: &client.FakeAPI{
			Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
			Channels_: []*model.Channel{{Id: "c1", Name: "general", DisplayName: "General", Type: model.ChannelTypeOpen}},
		},
		outputMode: "ai",
		userID:     "u1",
	}
	var buf bytes.Buffer
	if err := runChannelList(app, "eng", "default", &buf); err != nil {
		t.Fatalf("runChannelList: %v", err)
	}
	if !strings.Contains(buf.String(), "general") {
		t.Errorf("missing channel:\n%s", buf.String())
	}
}

func TestChannelList_HidesDMsByDefault_LabelsWhenShown(t *testing.T) {
	dm := &model.Channel{Id: "d1", Name: "u1__u2", Type: model.ChannelTypeDirect}
	pub := &model.Channel{Id: "c1", Name: "general", DisplayName: "General", Type: model.ChannelTypeOpen}

	base := func() *client.FakeAPI {
		return &client.FakeAPI{
			Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
			Channels_: []*model.Channel{pub, dm},
			Users_:    []*model.User{{Id: "u2", Username: "bob"}},
		}
	}

	// default: DM hidden
	app := &appContext{api: base(), outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runChannelList(app, "eng", "default", &buf); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "u1__u2") || strings.Contains(buf.String(), "d1") {
		t.Errorf("DM should be hidden by default:\n%s", buf.String())
	}

	// all: DM shown, labeled with the other user's username
	app = &appContext{api: base(), outputMode: "ai", userID: "u1"}
	buf.Reset()
	if err := runChannelList(app, "eng", "all", &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "@bob") {
		t.Errorf("DM should be labeled with @bob:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "u1__u2") {
		t.Errorf("raw DM name should be replaced:\n%s", buf.String())
	}
}

func TestChannelSearch(t *testing.T) {
	app := &appContext{
		api: &client.FakeAPI{
			Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
			Channels_: []*model.Channel{{Id: "c9", Name: "town-square", DisplayName: "Town Square", Type: model.ChannelTypeOpen}},
		},
		outputMode: "ai",
		userID:     "u1",
	}
	var buf bytes.Buffer
	if err := runChannelSearch(app, "eng", "town", &buf); err != nil {
		t.Fatalf("runChannelSearch: %v", err)
	}
	if !strings.Contains(buf.String(), "town-square") {
		t.Errorf("missing searched channel:\n%s", buf.String())
	}
}

func TestChannelList_BotLabel(t *testing.T) {
	dm := &model.Channel{Id: "d1", Name: "u1__u3", Type: model.ChannelTypeDirect}
	pub := &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}
	app := &appContext{
		api: &client.FakeAPI{
			Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
			Channels_: []*model.Channel{pub, dm},
			Users_:    []*model.User{{Id: "u3", Username: "botuser"}},
		},
		outputMode: "ai",
		userID:     "u1",
		botIDs:     []string{"u3"},
	}
	var buf bytes.Buffer
	if err := runChannelList(app, "eng", "all", &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "🤖@botuser") {
		t.Errorf("bot DM should be labeled with 🤖@botuser:\n%s", buf.String())
	}
}
