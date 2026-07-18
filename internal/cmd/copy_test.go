package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestCopy(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "pbcopy")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat >>"+filepath.Join(dir, "clipboard")+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	fake := &fakeAPI{
		thread:   &model.PostList{Posts: map[string]*model.Post{"p1": {Id: "p1", ChannelId: "c1"}}},
		resolved: &model.Channel{Id: "c1", TeamId: "t1", Name: "general"},
		teams:    []*model.Team{{Id: "t1", Name: "eng"}},
	}
	app := &appContext{api: fake, userID: "u1", outputMode: "ai"}

	url, err := resolveOpenURL(context.Background(), app, "p1")
	if err != nil {
		t.Fatalf("resolveOpenURL: %v", err)
	}

	if err := copyToClipboard(url); err != nil {
		t.Fatalf("copyToClipboard: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "clipboard"))
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	if !strings.Contains(got, "/eng/pl/p1") {
		t.Errorf("clipboard = %q, want .../eng/pl/p1", got)
	}
}

func TestCopyToClipboard(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "pbcopy")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat >>"+filepath.Join(dir, "clipboard")+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

	if err := copyToClipboard("test-url"); err != nil {
		t.Fatalf("copyToClipboard: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "clipboard"))
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	if got != "test-url" {
		t.Errorf("clipboard = %q, want test-url", got)
	}
}
