package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

// TestThreadListSince_CallCount proves that `thread list --since` issues a
// single batched UserThreads call for the whole team instead of one call per
// matching thread (an N+1 regression).
func TestThreadListSince_CallCount(t *testing.T) {
	const total, inWindow = 50, 5
	now := time.Now().UnixMilli()
	threads := make([]*model.ThreadResponse, 0, total)
	for i := 0; i < total; i++ {
		last := int64(0) // far outside the since window (epoch)
		if i < inWindow {
			last = now
		}
		threads = append(threads, &model.ThreadResponse{
			PostId:      fmt.Sprintf("p%d", i),
			ReplyCount:  1,
			LastReplyAt: last,
			Post: &model.Post{
				Id:        fmt.Sprintf("p%d", i),
				Message:   "root",
				UserId:    "u2",
				ChannelId: "c1",
				CreateAt:  last,
			},
		})
	}
	fake := &client.FakeAPI{
		Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
		Threads_:  &model.Threads{Threads: threads},
		Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen},
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	if err := runThreadList(app, threadListOpts{since: "24h", team: "eng"}, io.Discard); err != nil {
		t.Fatalf("runThreadList: %v", err)
	}
	if got := fake.UserThreadsCalls_; got != 1 {
		t.Errorf("UserThreads calls = %d, want 1 (one batch call, not one per matching thread)", got)
	}
	if fake.UserThreadsSince <= 0 {
		t.Errorf("UserThreadsSince = %d, want the parsed --since value passed through (> 0)", fake.UserThreadsSince)
	}
}

// TestDMListSince_CallCount proves that `dm list --since` issues exactly one
// ChannelsForUser and one Search per team instead of one per DM/GM channel (an
// N+1 regression).
func TestDMListSince_CallCount(t *testing.T) {
	const teamCount, dmChannels, publicChannels = 2, 30, 4
	now := time.Now().UnixMilli()
	channels := make([]*model.Channel, 0, dmChannels+publicChannels)
	for i := 0; i < dmChannels; i++ {
		channels = append(channels, &model.Channel{Id: fmt.Sprintf("dm%d", i), Name: "u1__u2", Type: model.ChannelTypeDirect})
	}
	for i := 0; i < publicChannels; i++ {
		channels = append(channels, &model.Channel{Id: fmt.Sprintf("c%d", i), Name: "public", Type: model.ChannelTypeOpen})
	}
	posts := &model.PostList{Order: make([]string, 0, dmChannels), Posts: make(map[string]*model.Post, dmChannels)}
	for i := 0; i < dmChannels; i++ {
		channelID := fmt.Sprintf("dm%d", i)
		postID := fmt.Sprintf("p%d", i)
		posts.Order = append(posts.Order, postID)
		posts.Posts[postID] = &model.Post{Id: postID, Message: "hello", UserId: "u2", ChannelId: channelID, CreateAt: now}
	}
	fake := &client.FakeAPI{
		Teams_:    []*model.Team{{Id: "t1", Name: "a"}, {Id: "t2", Name: "b"}},
		Channels_: channels,
		Posts_:    posts,
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	if err := runDMList(app, dmOpts{since: "24h", limit: 100}, io.Discard); err != nil {
		t.Fatalf("runDMList: %v", err)
	}
	if got := fake.ChannelsForUserCalls_; got != teamCount {
		t.Errorf("ChannelsForUser calls = %d, want %d (one per team, not one per DM channel)", got, teamCount)
	}
	if got := fake.SearchCalls_; got != teamCount {
		t.Errorf("Search calls = %d, want %d (one per team, not one per DM channel)", got, teamCount)
	}
}

// BenchmarkThreadListSince measures the full render path of `thread list
// --since` against a 1000-thread fake (half in window).
func BenchmarkThreadListSince(b *testing.B) {
	b.ReportAllocs()
	const total = 1000
	now := time.Now().UnixMilli()
	threads := make([]*model.ThreadResponse, 0, total)
	for i := 0; i < total; i++ {
		last := int64(0)
		if i%2 == 0 {
			last = now
		}
		threads = append(threads, &model.ThreadResponse{
			PostId:      fmt.Sprintf("p%d", i),
			ReplyCount:  1,
			LastReplyAt: last,
			Post: &model.Post{
				Id:        fmt.Sprintf("p%d", i),
				Message:   "root",
				UserId:    "u2",
				ChannelId: "c1",
				CreateAt:  last,
			},
		})
	}
	fake := &client.FakeAPI{
		Teams_:    []*model.Team{{Id: "t1", Name: "eng"}},
		Threads_:  &model.Threads{Threads: threads},
		Resolved_: &model.Channel{Id: "c1", Name: "general", Type: model.ChannelTypeOpen},
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	opts := threadListOpts{since: "24h", team: "eng", limit: total}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := runThreadList(app, opts, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDMListSince measures the full render path of `dm list --since`
// against 2 teams, 500 DM channels, and 1000 posts.
func BenchmarkDMListSince(b *testing.B) {
	b.ReportAllocs()
	const dmTotal, postTotal = 500, 1000
	now := time.Now().UnixMilli()
	channels := make([]*model.Channel, 0, dmTotal)
	for i := 0; i < dmTotal; i++ {
		channels = append(channels, &model.Channel{Id: fmt.Sprintf("dm%d", i), Name: "u1__u2", Type: model.ChannelTypeDirect})
	}
	posts := &model.PostList{Order: make([]string, 0, postTotal), Posts: make(map[string]*model.Post, postTotal)}
	for i := 0; i < postTotal; i++ {
		postID := fmt.Sprintf("p%d", i)
		posts.Order = append(posts.Order, postID)
		posts.Posts[postID] = &model.Post{Id: postID, Message: "hello", UserId: "u2", ChannelId: fmt.Sprintf("dm%d", i%dmTotal), CreateAt: now}
	}
	fake := &client.FakeAPI{
		Teams_:    []*model.Team{{Id: "t1", Name: "a"}, {Id: "t2", Name: "b"}},
		Channels_: channels,
		Posts_:    posts,
		Users_:    []*model.User{{Id: "u2", Username: "bob"}},
	}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1", previewLen: 140}
	opts := dmOpts{since: "24h", limit: postTotal}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := runDMList(app, opts, io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

// TestLiveServerPerf is a manual, env-gated wall-clock harness against a real
// server. It never fails on network errors — it logs and moves on.
func TestLiveServerPerf(t *testing.T) {
	url := os.Getenv("MMRUN_URL")
	token := os.Getenv("MMRUN_TOKEN")
	if url == "" || token == "" {
		t.Skip("set MMRUN_URL and MMRUN_TOKEN to run live perf test")
	}
	cl := client.NewWithToken(url, token, nil)
	me, err := cl.Me(context.Background())
	if err != nil {
		t.Logf("Me() against %s failed: %v — cannot build appContext", url, err)
		t.Skip("live perf test needs a working authenticated session")
	}
	app := &appContext{
		api:          cl,
		outputMode:   "ai",
		userID:       me.Id,
		username:     me.Username,
		previewLen:   140,
		defaultLimit: 50,
	}
	start := time.Now()
	if err := runThreadList(app, threadListOpts{since: "24h"}, io.Discard); err != nil {
		t.Logf("runThreadList live: %v", err)
	}
	t.Logf("runThreadList(since=24h) wall-clock: %v", time.Since(start))

	start = time.Now()
	if err := runDMList(app, dmOpts{since: "24h", limit: 50}, io.Discard); err != nil {
		t.Logf("runDMList live: %v", err)
	}
	t.Logf("runDMList(since=24h) wall-clock: %v", time.Since(start))
}
