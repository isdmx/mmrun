package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/isdmx/mmrun/internal/client"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestUserSearch(t *testing.T) {
	app := &appContext{
		api: &client.FakeAPI{
			Users_: []*model.User{{Id: "u2", Username: "bob", FirstName: "Bob", LastName: "Jones", Email: "b@x.com"}},
		},
		outputMode: "ai",
	}
	var buf bytes.Buffer
	if err := runUserSearch(app, "bob", "", &buf); err != nil {
		t.Fatalf("runUserSearch: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"@bob", "Bob Jones", "b@x.com", "u2"} {
		if !strings.Contains(out, want) {
			t.Errorf("user search output missing %q:\n%s", want, out)
		}
	}
}
