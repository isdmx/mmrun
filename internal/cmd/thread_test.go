package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestThreadRead_MarkRead(t *testing.T) {
	fake := &client.FakeAPI{
		Teams_: []*model.Team{{Id: "t1", Name: "eng"}},
		Thread_: &model.PostList{
			Order: []string{"p1"},
			Posts: map[string]*model.Post{"p1": {Id: "p1", Message: "root", UserId: "u2", ChannelId: "c1", CreateAt: 1000}},
		},
		Resolved_: &model.Channel{Id: "c1", Name: "general", TeamId: "t1", Type: model.ChannelTypeOpen},
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runThreadRead(app, "p1", 0, true, "", "", "", true, &buf); err != nil {
		t.Fatalf("thread read mark-read: %v", err)
	}
	if fake.ReadThread_ != "p1" {
		t.Errorf("UpdateThreadRead called with %q, want p1", fake.ReadThread_)
	}
}

func TestThreadList_ColumnsFilter(t *testing.T) {
	th := &model.Threads{Threads: []*model.ThreadResponse{
		{PostId: "p1", ReplyCount: 2, Post: &model.Post{Id: "p1", Message: "root", UserId: "u2", ChannelId: "c1"}},
	}}
	app := &appContext{
		api:        &client.FakeAPI{Teams_: []*model.Team{{Id: "t1", Name: "eng"}}, Threads_: th, Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen}, Users_: []*model.User{{Id: "u2", Username: "bob"}}},
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
		api: &client.FakeAPI{
			Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
			Threads_:  th,
			Resolved_: &model.Channel{Id: "c1", Name: "general", DisplayName: "General"},
			Users_:    []*model.User{{Id: "u2", Username: "bob"}},
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

func TestThreadList_MultiTeamNoError(t *testing.T) {
	th := &model.Threads{Threads: []*model.ThreadResponse{
		{PostId: "p1", ReplyCount: 1, Post: &model.Post{Id: "p1", Message: "root", UserId: "u2", ChannelId: "c1"}},
	}}
	fake := &client.FakeAPI{
		Teams_:    []*model.Team{{Id: "t1", Name: "sberdevices"}, {Id: "t2", Name: "gi"}},
		Threads_:  th,
		Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen},
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runThreadList(app, threadListOpts{limit: 30}, &buf); err != nil {
		t.Fatalf("multi-team thread list should not error: %v", err)
	}
	if fake.UserThreadsCalls_ != 2 {
		t.Errorf("UserThreads calls = %d, want 2 (one per team)", fake.UserThreadsCalls_)
	}
}

func TestThreadReadRecent_FiltersBySince(t *testing.T) {
	fake := &client.FakeAPI{
		Teams_: []*model.Team{{Id: "t1", Name: "eng"}},
		Threads_: &model.Threads{Threads: []*model.ThreadResponse{
			{PostId: "p1", LastReplyAt: 5000, Post: &model.Post{Id: "p1", Message: "old root", UserId: "u2", ChannelId: "c1", CreateAt: 1000}},
		}},
		Thread_: &model.PostList{
			Order: []string{"p1", "p2"},
			Posts: map[string]*model.Post{
				"p1": {Id: "p1", Message: "old root", UserId: "u2", ChannelId: "c1", CreateAt: 1000},
				"p2": {Id: "p2", Message: "recent reply", UserId: "u2", ChannelId: "c1", CreateAt: 5000},
			},
		},
		Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen},
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runThreadReadRecent(app, "", 2000, 30, "", "", "", true, &buf); err != nil {
		t.Fatalf("runThreadReadRecent: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "recent reply") {
		t.Errorf("output missing recent reply:\n%s", out)
	}
	if strings.Contains(out, "old root") {
		t.Errorf("output should exclude posts before --since:\n%s", out)
	}
	if fake.PostThreadCalls_ != 1 {
		t.Errorf("PostThread calls = %d, want 1", fake.PostThreadCalls_)
	}
}

func TestThreadReadRecent_ParallelFetch(t *testing.T) {
	fake := &client.FakeAPI{
		Teams_: []*model.Team{{Id: "t1", Name: "eng"}},
		Threads_: &model.Threads{Threads: []*model.ThreadResponse{
			{PostId: "p1", Post: &model.Post{Id: "p1", UserId: "u2", ChannelId: "c1", CreateAt: 5000}},
			{PostId: "p2", Post: &model.Post{Id: "p2", UserId: "u2", ChannelId: "c1", CreateAt: 5000}},
			{PostId: "p3", Post: &model.Post{Id: "p3", UserId: "u2", ChannelId: "c1", CreateAt: 5000}},
		}},
		Thread_:   &model.PostList{Order: []string{}, Posts: map[string]*model.Post{}},
		Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen},
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	var buf bytes.Buffer
	if err := runThreadReadRecent(app, "", 1000, 30, "", "", "", true, &buf); err != nil {
		t.Fatalf("runThreadReadRecent: %v", err)
	}
	if fake.PostThreadCalls_ != 3 {
		t.Errorf("PostThread calls = %d, want 3 (one per active thread)", fake.PostThreadCalls_)
	}
	if fake.UserThreadsCalls_ != 1 {
		t.Errorf("UserThreads calls = %d, want 1", fake.UserThreadsCalls_)
	}
}

func TestThreadFollow_Root(t *testing.T) {
	fake := &client.FakeAPI{
		Thread_: &model.PostList{
			Order: []string{"p1"},
			Posts: map[string]*model.Post{
				"p1": {Id: "p1", Message: "root", UserId: "u2", ChannelId: "c1", CreateAt: 1000},
			},
		},
		Resolved_: &model.Channel{Id: "c1", Name: "general", TeamId: "t1", Type: model.ChannelTypeOpen},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runThreadFollow(app, "p1", true, &buf); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if fake.Followed_ != "p1" {
		t.Errorf("Followed_ = %q, want p1", fake.Followed_)
	}
}

func TestThreadUnfollow_ResolvesReply(t *testing.T) {
	fake := &client.FakeAPI{
		Thread_: &model.PostList{
			Order: []string{"p1", "p2"},
			Posts: map[string]*model.Post{
				"p1": {Id: "p1", Message: "root", UserId: "u2", ChannelId: "c1", CreateAt: 1000},
				"p2": {Id: "p2", Message: "reply", UserId: "u3", ChannelId: "c1", RootId: "p1", CreateAt: 2000},
			},
		},
		Resolved_: &model.Channel{Id: "c1", Name: "general", TeamId: "t1", Type: model.ChannelTypeOpen},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runThreadFollow(app, "p2", false, &buf); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if fake.Unfollowed_ != "p1" {
		t.Errorf("Unfollowed_ = %q, want p1 (resolved root)", fake.Unfollowed_)
	}
}

func TestThreadUnfollow_RequiresYes(t *testing.T) {
	mode := "auto"
	var buf bytes.Buffer
	if err := runThreadFollowCmd(&mode, []string{"p1"}, false, false, &buf); err == nil {
		t.Error("unfollow without --yes should error")
	}
}
