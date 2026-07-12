package cmd

import (
	"testing"
)

func TestAppContext_UsesFake(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai"}
	got := app.api
	if got == nil {
		t.Fatal("api not wired")
	}
}
