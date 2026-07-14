package cmd

import (
	"bytes"
	"testing"
)

func TestReact(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runReact(app, "p1", "rocket", &buf); err != nil {
		t.Fatalf("react: %v", err)
	}
	if fake.reacted != "rocket" {
		t.Errorf("reacted emoji = %q, want rocket", fake.reacted)
	}
}

func TestReact_StripsColons(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runReact(app, "p1", ":rocket:", &buf); err != nil {
		t.Fatal(err)
	}
	if fake.reacted != "rocket" {
		t.Errorf("reacted emoji not stripped: %q", fake.reacted)
	}
}

func TestUnreact_RequiresYes(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runUnreact(app, "p1", "rocket", false, &buf); err == nil {
		t.Error("unreact without --yes should error")
	}
	if err := runUnreact(app, "p1", "rocket", true, &buf); err != nil {
		t.Errorf("unreact with --yes: %v", err)
	}
	if fake.unreacted != "rocket" {
		t.Errorf("unreacted emoji = %q, want rocket", fake.unreacted)
	}
}
