package output

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type treeBlockRenderer struct {
	color      bool
	theme      Theme
	timeFormat string
}

var _ Renderer = treeBlockRenderer{}

func (t treeBlockRenderer) Render(w io.Writer, r Result) error {
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
		if err := t.renderRow(w, row); err != nil {
			return err
		}
	}
	return nil
}

func (t treeBlockRenderer) renderRow(w io.Writer, row Row) error {
	isReply := row["root_id"] != ""
	prefix := "●"
	if isReply {
		prefix = "  ↳"
	}
	ts := row["time"]
	if t.timeFormat == "relative" {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			ts = reltime(parsed)
		}
	}
	if _, err := fmt.Fprintf(w, "%s %s · %s\n", prefix, row["user"], ts); err != nil {
		return err
	}
	indent := "  "
	if isReply {
		indent = "  │ "
	}
	for _, line := range strings.Split(row["message"], "\n") {
		if _, err := fmt.Fprintf(w, "%s%s\n", indent, line); err != nil {
			return err
		}
	}
	meta := chatMeta(row)
	if meta == "" {
		_, err := fmt.Fprintln(w)
		return err
	}
	if t.color {
		meta = "\x1b[2m" + meta + ansiReset
	}
	_, err := fmt.Fprintf(w, "  %s\n\n", meta)
	return err
}
