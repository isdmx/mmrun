package cmd

import (
	"context"
	"testing"

	"github.com/isdmx/mmrun/internal/client"
)

func TestStatusSet(t *testing.T) {
	fake := &client.FakeAPI{}
	app := &appContext{api: fake, userID: "u1"}
	if err := app.api.UpdateStatus(context.Background(), app.userID, "dnd"); err != nil {
		t.Fatal(err)
	}
	if err := app.api.UpdateCustomStatus(context.Background(), app.userID, "lunch", "out"); err != nil {
		t.Fatal(err)
	}
}
