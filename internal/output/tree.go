package output

import (
	"fmt"
	"io"
)

type treeRenderer struct{ color bool }

func (t treeRenderer) Render(w io.Writer, r Result) error {
	if r.Title != "" {
		title := r.Title
		if t.color {
			title = ansiBold + title + ansiReset
		}
		if _, err := fmt.Fprintln(w, title); err != nil {
			return err
		}
	}
	for _, row := range r.Rows {
		isReply := row["root_id"] != ""
		prefix := "●"
		if isReply {
			prefix = "  ↳"
		}
		user := row["user"]
		if user == "" {
			user = row["user_id"]
		}
		msg := row["message"]
		if _, err := fmt.Fprintf(w, "%s %s · %s\n", prefix, user, msg); err != nil {
			return err
		}
	}
	return nil
}
