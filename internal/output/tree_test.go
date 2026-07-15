package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTreeRenderer(t *testing.T) {
	r := treeRenderer{color: false}
	var buf bytes.Buffer
	res := Result{
		Title:   "Test",
		Columns: []string{"user", "root_id", "message"},
		Rows: []Row{
			{"user": "alice", "root_id": "", "message": "root post"},
			{"user": "bob", "root_id": "r1", "message": "reply to first"},
			{"user": "charlie", "root_id": "", "message": "second root"},
		},
	}
	if err := r.Render(&buf, res); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "●") || !strings.Contains(out, "↳") {
		t.Errorf("tree should contain root/reply markers:\n%s", out)
	}
}
