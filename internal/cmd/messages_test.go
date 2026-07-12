package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestRead_IncludesActionableColumns(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{
			"p1": {Id: "p1", Message: "line one\nline two\twith tabs", UserId: "u2", ChannelId: "c1", RootId: "r0", CreateAt: 1000},
		},
	}
	app := &appContext{
		api: &fakeAPI{
			resolved: &model.Channel{Id: "c1", Name: "general", DisplayName: "General", TeamId: "t1"},
			teams:    []*model.Team{{Id: "t1", Name: "eng"}},
			users:    []*model.User{{Id: "u2", Username: "bob"}},
			posts:    pl,
		},
		outputMode: "ai",
	}
	var buf bytes.Buffer
	if err := runRead(app, "eng/general", readOpts{limit: 50}, &buf); err != nil {
		t.Fatalf("runRead: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"post_id=p1",
		"root_id=r0",
		"user=bob",                    // resolved username, not the ID
		"channel=General",             // resolved channel name
		"pl/p1",                       // permalink
		"line one line two with tabs", // whitespace collapsed to single line
	} {
		if !strings.Contains(out, want) {
			t.Errorf("read output missing %q:\n%s", want, out)
		}
	}
}

func TestPreview_CollapsesAndTruncates(t *testing.T) {
	got := preview("a\n\nb\tc   d", 100)
	if got != "a b c d" {
		t.Errorf("preview collapse = %q, want %q", got, "a b c d")
	}
	long := strings.Repeat("x", 200)
	trunc := preview(long, 10)
	if trunc != strings.Repeat("x", 10)+"…" {
		t.Errorf("preview truncate = %q", trunc)
	}
}

func TestFileSummary(t *testing.T) {
	if got := fileSummary(&model.Post{}); got != "" {
		t.Errorf("no files = %q, want empty", got)
	}
	if got := fileSummary(&model.Post{FileIds: []string{"a", "b"}}); got != "2" {
		t.Errorf("count only = %q, want 2", got)
	}
	withMeta := &model.Post{
		FileIds:  []string{"a", "b"},
		Metadata: &model.PostMetadata{Files: []*model.FileInfo{{Name: "one.txt"}, {Name: "two.pdf"}}},
	}
	if got := fileSummary(withMeta); got != "2: one.txt, two.pdf" {
		t.Errorf("with names = %q, want %q", got, "2: one.txt, two.pdf")
	}
}

func TestChannelLabel_DirectMessage(t *testing.T) {
	app := &appContext{
		api:    &fakeAPI{resolved: &model.Channel{Id: "d1", Name: "u1__u2", Type: model.ChannelTypeDirect}, users: []*model.User{{Id: "u2", Username: "bob"}}},
		userID: "u1",
	}
	got := channelLabel(context.Background(), app, "d1", map[string]string{})
	if got != "@bob" {
		t.Errorf("DM label = %q, want @bob", got)
	}
}

func TestChannelLabel_SelfDM(t *testing.T) {
	app := &appContext{
		api:    &fakeAPI{resolved: &model.Channel{Id: "d1", Name: "u1__u1", Type: model.ChannelTypeDirect}},
		userID: "u1",
	}
	got := channelLabel(context.Background(), app, "d1", map[string]string{})
	if got != "you" {
		t.Errorf("self-DM label = %q, want you", got)
	}
}
