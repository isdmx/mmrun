package cmd

import (
	"context"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestCompleteChannels(t *testing.T) {
	fake := &fakeAPI{channels: []*model.Channel{
		{Name: "general", DisplayName: "General", Type: model.ChannelTypeOpen},
		{Name: "random", DisplayName: "Random", Type: model.ChannelTypeOpen},
		{Name: "u1__u2", Type: model.ChannelTypeDirect},
	}, me: &model.User{Username: "alice"}}
	app := &appContext{api: fake, userID: "u1"}
	got := completeChannelCompletions(app)
	if len(got) == 0 {
		t.Fatal("expected channels")
	}
	hasSelf, hasDM := false, false
	for _, g := range got {
		if g == "@alice" {
			hasSelf = true
		}
		if g == "@u2" {
			hasDM = true
		}
	}
	if !hasSelf {
		t.Error("self-completion (@alice) should always be included")
	}
	if !hasDM {
		t.Error("DM should complete as @username or @id fallback")
	}
}

func TestCompleteTeams(t *testing.T) {
	fake := &fakeAPI{teams: []*model.Team{{Name: "eng"}, {Name: "ops"}}}
	app := &appContext{api: fake, userID: "u1"}
	got := completeTeamCompletions(app)
	if len(got) != 2 || got[0] != "eng" || got[1] != "ops" {
		t.Errorf("teams = %v", got)
	}
}

func TestCompletePostIDs(t *testing.T) {
	fake := &fakeAPI{threads: &model.Threads{Threads: []*model.ThreadResponse{
		{PostId: "p1"},
		{PostId: "p2"},
	}}}
	app := &appContext{api: fake, userID: "u1"}
	got := completePostIDCompletions(app)
	if len(got) != 2 || got[0] != "p1" || got[1] != "p2" {
		t.Errorf("post IDs = %v", got)
	}
}

func TestResolveSelfCompletion_UsesUsername(t *testing.T) {
	app := &appContext{username: "alice"}
	got := resolveSelfCompletion(context.Background(), app)
	if got != "@alice" {
		t.Errorf("resolveSelf = %q, want @alice", got)
	}
}

func TestResolveSelfCompletion_FallsBackToGetMe(t *testing.T) {
	fake := &fakeAPI{me: &model.User{Username: "bob"}}
	app := &appContext{api: fake}
	got := resolveSelfCompletion(context.Background(), app)
	if got != "@bob" {
		t.Errorf("resolveSelf fallback = %q, want @bob", got)
	}
}
