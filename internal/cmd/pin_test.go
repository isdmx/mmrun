package cmd

import (
	"bytes"
	"context"
	"testing"
)

func TestPin(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	if err := app.api.PinPost(context.Background(), "p1"); err != nil {
		t.Fatalf("pin: %v", err)
	}
	if fake.pinned != "p1" {
		t.Error("PinPost not called")
	}
}

func TestUnpin_RequiresYes(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runUnpin(app, "p1", false, &buf); err == nil {
		t.Error("unpin without --yes should error")
	}
	if err := runUnpin(app, "p1", true, &buf); err != nil {
		t.Errorf("unpin with --yes: %v", err)
	}
	if fake.unpinned != "p1" {
		t.Error("UnpinPost should be called")
	}
}
