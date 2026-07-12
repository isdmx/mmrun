package cmd

import (
	"encoding/json"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
)

func TestPostedEventToRow(t *testing.T) {
	post := &model.Post{Id: "p1", Message: "hi there", UserId: "u9", CreateAt: 4000}
	b, _ := json.Marshal(post)
	data := map[string]any{"post": string(b), "channel_name": "general"}

	row, ok := postedEventToRow("c1", "posted", data)
	if !ok {
		t.Fatal("expected a row for posted event in matching channel")
	}
	if row["message"] != "hi there" || row["user_id"] != "u9" {
		t.Errorf("unexpected row: %+v", row)
	}
}

func TestPostedEventToRow_IgnoresOtherEvents(t *testing.T) {
	if _, ok := postedEventToRow("c1", "typing", map[string]any{}); ok {
		t.Error("typing events should be ignored")
	}
}
