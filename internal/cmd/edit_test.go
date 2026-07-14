package cmd

import (
	"bytes"
	"testing"
)

func TestEdit(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runEdit(app, "p1", "updated", &buf); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if fake.patched == nil || fake.patched.Message != "updated" {
		t.Error("PatchPost should be called with updated message")
	}
}

func TestDelete_RequiresYes(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai", userID: "u1"}
	var buf bytes.Buffer
	if err := runDelete(app, "p1", false, &buf); err == nil {
		t.Error("delete without --yes should error")
	}
	if err := runDelete(app, "p1", true, &buf); err != nil {
		t.Errorf("delete with --yes: %v", err)
	}
	if fake.deleted != "p1" {
		t.Error("DeletePost should be called")
	}
}
