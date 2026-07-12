package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestGetMe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/users/me" {
			_ = json.NewEncoder(w).Encode(&model.User{Id: "u1", Username: "alice"})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := NewWithToken(srv.URL, "tok")
	u, err := c.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("username = %q, want alice", u.Username)
	}
}

func TestResolveChannel_ByTeamAndName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/teams/name/eng":
			_ = json.NewEncoder(w).Encode(&model.Team{Id: "t1", Name: "eng"})
		case "/api/v4/teams/t1/channels/name/general":
			_ = json.NewEncoder(w).Encode(&model.Channel{Id: "c1", Name: "general", TeamId: "t1"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewWithToken(srv.URL, "tok")
	ch, err := c.ResolveChannel(context.Background(), "eng/general", "")
	if err != nil {
		t.Fatalf("ResolveChannel: %v", err)
	}
	if ch.Id != "c1" {
		t.Errorf("channel id = %q, want c1", ch.Id)
	}
}
