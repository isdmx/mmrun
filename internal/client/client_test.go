package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestStatusCode(t *testing.T) {
	if got := StatusCode(nil); got != 0 {
		t.Errorf("nil = %d, want 0", got)
	}
	appErr := &model.AppError{StatusCode: 401, Message: "unauthorized"}
	if got := StatusCode(appErr); got != 401 {
		t.Errorf("401 = %d", got)
	}
	plainErr := errors.New("network error")
	if got := StatusCode(plainErr); got != 0 {
		t.Errorf("plain = %d, want 0", got)
	}
}

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
	ch, err := c.ResolveChannel(context.Background(), "eng/general", "", "self")
	if err != nil {
		t.Fatalf("ResolveChannel: %v", err)
	}
	if ch.Id != "c1" {
		t.Errorf("channel id = %q, want c1", ch.Id)
	}
}

func TestResolveChannel_DirectMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/users/username/bob":
			_ = json.NewEncoder(w).Encode(&model.User{Id: "u2", Username: "bob"})
		case "/api/v4/channels/direct":
			_ = json.NewEncoder(w).Encode(&model.Channel{Id: "dm1", Type: model.ChannelTypeDirect})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewWithToken(srv.URL, "tok")
	ch, err := c.ResolveChannel(context.Background(), "@bob", "", "u1")
	if err != nil {
		t.Fatalf("ResolveChannel DM: %v", err)
	}
	if ch.Id != "dm1" {
		t.Errorf("channel id = %q, want dm1", ch.Id)
	}
}

func TestResolveChannel_ByEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/users/email/alice@example.com":
			_ = json.NewEncoder(w).Encode(&model.User{Id: "u2", Username: "alice"})
		case "/api/v4/channels/direct":
			_ = json.NewEncoder(w).Encode(&model.Channel{Id: "dm1", Type: model.ChannelTypeDirect})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewWithToken(srv.URL, "tok")
	ch, err := c.ResolveChannel(context.Background(), "alice@example.com", "", "u1")
	if err != nil {
		t.Fatalf("email resolve: %v", err)
	}
	if ch.Id != "dm1" {
		t.Errorf("channel id = %q, want dm1", ch.Id)
	}
}

func TestResolveChannel_TildeChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/users/u1/teams":
			_ = json.NewEncoder(w).Encode([]*model.Team{{Id: "t1", Name: "eng"}})
		case "/api/v4/teams/t1/channels/name/general":
			_ = json.NewEncoder(w).Encode(&model.Channel{Id: "c1", Name: "general"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewWithToken(srv.URL, "tok")
	ch, err := c.ResolveChannel(context.Background(), "~general", "", "u1")
	if err != nil {
		t.Fatalf("tilde resolve: %v", err)
	}
	if ch.Id != "c1" {
		t.Errorf("channel id = %q, want c1", ch.Id)
	}
}

func TestResolveChannel_IDFallbackToUser(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/channels/ch266charidXXXXXXXXXXXXXXX":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(&model.AppError{StatusCode: 404})
		case "/api/v4/users/ch266charidXXXXXXXXXXXXXXX":
			_ = json.NewEncoder(w).Encode(&model.User{Id: "u9", Username: "target"})
		case "/api/v4/channels/direct":
			_ = json.NewEncoder(w).Encode(&model.Channel{Id: "dm1", Type: model.ChannelTypeDirect})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewWithToken(srv.URL, "tok")
	ch, err := c.ResolveChannel(context.Background(), "ch266charidXXXXXXXXXXXXXXX", "", "u1")
	if err != nil {
		t.Fatalf("id fallback: %v", err)
	}
	if ch.Id != "dm1" {
		t.Errorf("channel id = %q, want dm1", ch.Id)
	}
}

func TestResolveChannel_BareWordFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/teams/name/eng":
			_ = json.NewEncoder(w).Encode(&model.Team{Id: "t1", Name: "eng"})
		case "/api/v4/teams/t1/channels/name/nope":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(&model.AppError{StatusCode: 404})
		case "/api/v4/users/username/nope":
			_ = json.NewEncoder(w).Encode(&model.User{Id: "u2", Username: "nope"})
		case "/api/v4/channels/direct":
			_ = json.NewEncoder(w).Encode(&model.Channel{Id: "dm1", Type: model.ChannelTypeDirect})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewWithToken(srv.URL, "tok")
	ch, err := c.ResolveChannel(context.Background(), "nope", "eng", "u1")
	if err != nil {
		t.Fatalf("bare word fallback: %v", err)
	}
	if ch.Id != "dm1" {
		t.Errorf("channel id = %q, want dm1", ch.Id)
	}
}
