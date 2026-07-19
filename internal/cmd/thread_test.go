package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestThreadRead_MarkRead(t *testing.T) {
	fake := &fakeAPI{
		teams: []*model.Team{{Id: "t1", Name: "eng"}},
		thread: &model.PostList{
			Order: []string{"p1"},
			Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "root", UserId: "u2", ChannelId: "c1", CreateAt: 1000}},
		},
		resolved: &model.Channel{Id: "c1", Name: "general", TeamId: "t1", Type: model.ChannelTypeOpen},
		users:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runThreadRead(app, "p1", true, "", "", "", true, &buf); err != nil {
		t.Fatalf("thread read mark-read: %v", err)
	}
	if fake.readThread != "p1" {
		t.Errorf("UpdateThreadRead called with %q, want p1", fake.readThread)
	}
}

func TestThreadList_ColumnsFilter(t *testing.T) {
	th := &model.Threads{Threads: []*model.ThreadResponse{
		{PostId: "p1", ReplyCount: 2, Post: &model.Post{Id: "p1", Message: "root", UserId: "u2", ChannelId: "c1"}},
	}}
	app := &appContext{
		api:        &fakeAPI{teams: []*model.Team{{Id: "t1", Name: "eng"}}, threads: th, resolved: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, users: []*model.User{{Id: "u2", Username: "bob"}}},
		outputMode: "ai",
		userID:     "u1",
		previewLen: 140,
	}
	var buf bytes.Buffer
	if err := runThreadList(app, threadListOpts{limit: 30, columns: "user,replies,message"}, &buf); err != nil {
		t.Fatalf("runThreadList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "replies=2") || strings.Contains(out, "permalink=") {
		t.Errorf("columns not applied:\n%s", out)
	}
}

func TestThreadList_RendersFollowedThreads(t *testing.T) {
	th := &model.Threads{
		Threads: []*model.ThreadResponse{
			{
				PostId:        "p1",
				ReplyCount:    3,
				UnreadReplies: 1,
				LastReplyAt:   5000,
				Post:          &model.Post{Id: "p1", Message: "root of thread", UserId: "u2", ChannelId: "c1"},
			},
		},
	}
	app := &appContext{
		api: &fakeAPI{
			teams:    []*model.Team{{Id: "t1", Name: "eng"}},
			threads:  th,
			resolved: &model.Channel{Id: "c1", Name: "general", DisplayName: "General"},
			users:    []*model.User{{Id: "u2", Username: "bob"}},
		},
		outputMode: "ai",
		userID:     "u1",
		previewLen: 140,
	}
	var buf bytes.Buffer
	if err := runThreadList(app, threadListOpts{limit: 30}, &buf); err != nil {
		t.Fatalf("runThreadList: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"post_id=p1", "replies=3", "unread=1", "user=bob", "channel=General", "root of thread", "pl/p1"} {
		if !strings.Contains(out, want) {
			t.Errorf("thread output missing %q:\n%s", want, out)
		}
	}
}
