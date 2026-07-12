package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/isdmx/mmrun/internal/client"
	"github.com/mattermost/mattermost/server/public/model"
)

func TestPostedEventToRow(t *testing.T) {
	post := &model.Post{Id: "p1", Message: "hi there", UserId: "u9", CreateAt: 4000}
	b, _ := json.Marshal(post)
	data := map[string]any{"post": string(b), "channel_name": "general"}

	row, ok := postedEventToRow("c1", "posted", data)
	if !ok {
		t.Fatal("expected a row for posted event in matching channel")
	}
	if row["message"] != "hi there" || row["user_id"] != "u9" {
		t.Errorf("unexpected row: %+v", row)
	}
}

func TestPostedEventToRow_IgnoresOtherEvents(t *testing.T) {
	if _, ok := postedEventToRow("c1", "typing", map[string]any{}); ok {
		t.Error("typing events should be ignored")
	}
}

func TestRunTail_RendersStreamedPost(t *testing.T) {
	events := make(chan client.WSEvent, 1)
	f := &fakeAPI{
		resolved:     &model.Channel{Id: "c1"},
		streamEvents: events,
		streamErrs:   make(chan error, 1),
	}
	app := &appContext{api: f, outputMode: "ai"}

	post := &model.Post{Id: "p1", Message: "live msg", UserId: "u9", CreateAt: 4000, ChannelId: "c1"}
	b, _ := json.Marshal(post)
	events <- client.WSEvent{Event: "posted", Data: map[string]any{"post": string(b)}}

	ctx, cancel := context.WithCancel(context.Background())
	var buf bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- runTail(ctx, app, "eng/general", "", &buf) }()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	if !strings.Contains(buf.String(), "live msg") {
		t.Errorf("expected streamed message in output:\n%s", buf.String())
	}
}

func TestRunTail_SurfacesStreamError(t *testing.T) {
	f := &fakeAPI{resolved: &model.Channel{Id: "c1"}, streamErr: context.DeadlineExceeded}
	app := &appContext{api: f, outputMode: "ai"}
	var buf bytes.Buffer
	if err := runTail(context.Background(), app, "eng/general", "", &buf); err == nil {
		t.Error("expected StreamPosts error to be surfaced")
	}
}
