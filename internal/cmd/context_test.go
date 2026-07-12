package cmd

import (
	"testing"

	"github.com/isdmx/mmrun/internal/client"
)

func TestAppContext_UsesFake(t *testing.T) {
	fake := &fakeAPI{}
	app := &appContext{api: fake, outputMode: "ai"}
	var got client.API = app.api
	if got == nil {
		t.Fatal("api not wired")
	}
}
