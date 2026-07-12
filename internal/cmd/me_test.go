package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestMeCommand_RendersAccount(t *testing.T) {
	app := &appContext{
		api: &fakeAPI{
			me:     &model.User{Id: "u1", Username: "alice", Nickname: "Al", Email: "a@x.com"},
			status: &model.Status{Status: "online"},
		},
		outputMode: "ai",
	}
	var buf bytes.Buffer
	if err := runMe(app, &buf); err != nil {
		t.Fatalf("runMe: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"alice", "Al", "a@x.com", "online"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
