package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTreeBlockRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := treeBlockRenderer{theme: DarkTheme}
	res := Result{
		Columns: []string{"user", "time", "root_id", "post_id", "message"},
		Rows: []Row{
			{"user": "@alice", "root_id": "", "post_id": "p1", "message": "Root message"},
			{"user": "@bob", "root_id": "p1", "post_id": "p2", "message": "Reply to root"},
			{"user": "@charlie", "root_id": "", "post_id": "p3", "message": "Second root"},
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
