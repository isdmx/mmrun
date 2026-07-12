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
		previewLen: 140,
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

func TestParseSince(t *testing.T) {
	ms, err := parseSince("24h")
	if err != nil {
		t.Fatalf("parseSince duration: %v", err)
	}
	if ms <= 0 {
		t.Errorf("expected positive ms, got %d", ms)
	}
	if _, err := parseSince("2026-01-02T15:04:05Z"); err != nil {
		t.Errorf("parseSince RFC3339: %v", err)
	}
	if _, err := parseSince("not-a-time"); err == nil {
		t.Error("expected error for garbage input")
	}
}

func TestRead_ColumnsFilter(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "hello", UserId: "u2", ChannelId: "c1", CreateAt: 1000}},
	}
	app := &appContext{
		api:        &fakeAPI{resolved: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, posts: pl, users: []*model.User{{Id: "u2", Username: "bob"}}},
		outputMode: "ai",
		previewLen: 140,
	}
	var buf bytes.Buffer
	if err := runRead(app, "eng/general", readOpts{columns: "user,message"}, &buf); err != nil {
		t.Fatalf("runRead: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "user=bob") || !strings.Contains(out, "message=hello") {
		t.Errorf("missing selected columns:\n%s", out)
	}
	if strings.Contains(out, "post_id=") || strings.Contains(out, "permalink=") {
		t.Errorf("unselected columns present:\n%s", out)
	}
}

func TestRead_BadColumn(t *testing.T) {
	app := &appContext{
		api:        &fakeAPI{resolved: &model.Channel{Id: "c1"}, posts: &model.PostList{}},
		outputMode: "ai",
		previewLen: 140,
	}
	var buf bytes.Buffer
	if err := runRead(app, "eng/general", readOpts{columns: "bogus"}, &buf); err == nil {
		t.Error("expected error for unknown column")
	}
}

func TestRead_Thread(t *testing.T) {
	pl := &model.PostList{
		Order: []string{"p1"},
		Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "thread root", CreateAt: 10}},
	}
	app := &appContext{
		api:        &fakeAPI{thread: pl},
		outputMode: "ai",
		previewLen: 140,
	}
	var buf bytes.Buffer
	if err := runRead(app, "ignored", readOpts{thread: "p1"}, &buf); err != nil {
		t.Fatalf("runRead thread: %v", err)
	}
	if !strings.Contains(buf.String(), "thread root") {
		t.Errorf("missing thread post:\n%s", buf.String())
	}
}
