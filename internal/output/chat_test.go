package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestChatRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := chatRenderer{theme: DarkTheme}
	res := Result{
		Title:   "Test",
		Columns: []string{"user", "time", "root_id", "post_id", "permalink", "files", "reactions", "message"},
		Rows: []Row{
			{"user": "@alice", "time": "2026-01-02T15:04:05Z", "post_id": "p1", "message": "Hello\nWorld"},
			{"user": "@bob", "time": "2026-01-02T15:05:05Z", "root_id": "p1", "post_id": "p2", "message": "Reply"},
		},
	}
	if err := r.Render(&buf, res); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "@alice") || !strings.Contains(out, "Hello") {
		t.Error("chat output should contain user and message")
	}
	if !strings.Contains(out, "@bob") || !strings.Contains(out, "Reply") {
		t.Error("chat output should contain second row")
	}
	// Metadata should appear
	if !strings.Contains(out, "post_id=p1") || !strings.Contains(out, "post_id=p2") {
		t.Error("chat footer should contain post_id")
	}
}
