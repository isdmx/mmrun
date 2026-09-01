package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/isdmx/mmrun/internal/client"
)

func TestRead_IncludesActionableColumns(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{
			"p1": {Id: "p1", Message: "line one\nline two\twith tabs", UserId: "u2", ChannelId: "c1", RootId: "r0", CreateAt: 1000},
		},
	}
	app := &appContext{
		api: &client.FakeAPI{
			Resolved_: &model.Channel{Id: "c1", Name: "general", DisplayName: "General", TeamId: "t1"},
			Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
			Users_:    []*model.User{{Id: "u2", Username: "bob"}},
			Posts_:    pl,
		},
		outputMode: "ai",
		previewLen: 140,
	}
	var buf bytes.Buffer
	if err := runRead(app, "eng/general", readOpts{limit: 50}, &buf); err != nil {
		t.Fatalf("runRead: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"post_id=p1",
		"root_id=r0",
		"user=@bob",                   // resolved username with @ prefix
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
		api:    &client.FakeAPI{Resolved_: &model.Channel{Id: "d1", Name: "u1__u2", Type: model.ChannelTypeDirect}, Users_: []*model.User{{Id: "u2", Username: "bob"}}},
		userID: "u1",
	}
	got := channelLabel(context.Background(), app, "d1", map[string]string{})
	if got != "@bob" {
		t.Errorf("DM label = %q, want @bob", got)
	}
}

func TestResolveReactions(t *testing.T) {
	fake := &client.FakeAPI{Reactions_: []*model.Reaction{
		{PostId: "p1", EmojiName: "thumbsup", UserId: "u2"},
		{PostId: "p1", EmojiName: "thumbsup", UserId: "u3"},
		{PostId: "p1", EmojiName: "rocket", UserId: "u2"},
	}}
	app := &appContext{api: fake}
	posts := []*model.Post{{Id: "p1"}}
	got := resolveReactions(context.Background(), app, posts)
	if v, ok := got["p1"]; !ok || !strings.Contains(v, "thumbsup") || !strings.Contains(v, "2") {
		t.Errorf("reactions = %q, want :thumbsup: 2 :rocket: 1", v)
	}
}

func TestChannelLabel_SelfDM(t *testing.T) {
	app := &appContext{
		api:    &client.FakeAPI{Resolved_: &model.Channel{Id: "d1", Name: "u1__u1", Type: model.ChannelTypeDirect}},
		userID: "u1",
	}
	got := channelLabel(context.Background(), app, "d1", map[string]string{})
	if got != "you" {
		t.Errorf("self-DM label = %q, want you", got)
	}
}

func TestChannelLabel_GroupMessage(t *testing.T) {
	app := &appContext{
		api: &client.FakeAPI{
			Resolved_: &model.Channel{Id: "g1", Name: "hash", Type: model.ChannelTypeGroup},
			Members_:  model.ChannelMembers{{ChannelId: "g1", UserId: "u1"}, {ChannelId: "g1", UserId: "u3"}, {ChannelId: "g1", UserId: "u2"}},
			Users_:    []*model.User{{Id: "u2", Username: "bob"}, {Id: "u3", Username: "carol"}},
		},
		userID: "u1",
	}
	got := channelLabel(context.Background(), app, "g1", map[string]string{})
	if got != "@bob, @carol" {
		t.Errorf("group label = %q, want %q", got, "@bob, @carol")
	}
}

func TestChannelLabel_SelfOnlyGroup(t *testing.T) {
	app := &appContext{
		api: &client.FakeAPI{
			Resolved_: &model.Channel{Id: "g1", Name: "hash", Type: model.ChannelTypeGroup},
			Members_:  model.ChannelMembers{{ChannelId: "g1", UserId: "u1"}},
		},
		userID: "u1",
	}
	got := channelLabel(context.Background(), app, "g1", map[string]string{})
	if got != "you" {
		t.Errorf("self-only GM label = %q, want you", got)
	}
}

func TestStatusDots(t *testing.T) {
	fake := &client.FakeAPI{Statuses_: []*model.Status{
		{UserId: "u2", Status: "online"},
		{UserId: "u3", Status: "offline"},
	}}
	app := &appContext{api: fake}
	posts := []*model.Post{{UserId: "u2"}, {UserId: "u3"}}
	got := resolveStatuses(context.Background(), app, posts)
	if got["u2"] != "🟢" {
		t.Errorf("online = %q, want 🟢", got["u2"])
	}
	if got["u3"] != "⛔" {
		t.Errorf("offline = %q, want ⛔", got["u3"])
	}
}

func TestExtractLinks(t *testing.T) {
	msg := "check https://example.com/foo and https://bar.com/baz?q=1"
	links := extractLinks(msg)
	if len(links) != 2 || links[0] != "https://example.com/foo" || links[1] != "https://bar.com/baz?q=1" {
		t.Errorf("links = %v", links)
	}
}

func TestExtractLinks_NoLinks(t *testing.T) {
	if len(extractLinks("no links here")) != 0 {
		t.Error("expected no links")
	}
}

func TestReadHideChannel(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "hi", UserId: "u2", ChannelId: "c1", CreateAt: 1000}},
	}
	fake := &client.FakeAPI{Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, Posts_: pl, Users_: []*model.User{{Id: "u2", Username: "bob"}}}
	app := &appContext{api: fake, outputMode: "ai", previewLen: 140}
	var buf bytes.Buffer
	if err := runRead(app, "eng/general", readOpts{limit: 50}, &buf); err != nil {
		t.Fatalf("runRead: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "channel=general") || strings.Contains(out, "channel=General") {
		t.Error("read should hide channel column (redundant)")
	}
	if !strings.Contains(out, "user=@bob") {
		t.Error("user should have @ prefix")
	}
}

func TestPinnedColumn(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "hi", UserId: "u2", ChannelId: "c1", CreateAt: 1000, IsPinned: true}},
	}
	fake := &client.FakeAPI{Resolved_: &model.Channel{Id: "c1", Name: "g", Type: model.ChannelTypeOpen}, Posts_: pl, Users_: []*model.User{{Id: "u2", Username: "bob"}}}
	app := &appContext{api: fake, outputMode: "ai", previewLen: 140}
	columns, _ := resolveColumns(messageColumns, "")
	res := renderMessages(context.Background(), app, "Test", client.SortPosts(pl), "", false, columns, true, "")
	if res.Rows[0]["pinned"] != "📌" {
		t.Errorf("pinned = %q, want 📌", res.Rows[0]["pinned"])
	}
}
