package cmd

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestCompleteChannels(t *testing.T) {
	fake := &fakeAPI{channels: []*model.Channel{
		{Name: "general", DisplayName: "General", Type: model.ChannelTypeOpen},
		{Name: "random", DisplayName: "Random", Type: model.ChannelTypeOpen},
		{Name: "u1__u2", Type: model.ChannelTypeDirect},
	}}
	app := &appContext{api: fake, userID: "u1"}
	got := completeChannelCompletions(app)
	if len(got) == 0 {
		t.Fatal("expected channels")
	}
	hasDM := false
	for _, g := range got {
		if g == "@u2" {
			hasDM = true
		}
	}
	if !hasDM {
		t.Error("DM should complete as @username")
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
