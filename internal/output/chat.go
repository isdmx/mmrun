package output

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type chatRenderer struct {
	color      bool
	theme      Theme
	timeFormat string
	markdown   bool
}

var _ Renderer = chatRenderer{}

func (c chatRenderer) Render(w io.Writer, r Result) error {
	if r.Title != "" {
		title := r.Title
		if c.color {
			title = ansiBold + title + ansiReset
		}
		if _, err := fmt.Fprintln(w, title); err != nil {
			return err
		}
	}
	for _, row := range r.Rows {
		if err := c.renderRow(w, row); err != nil {
			return err
		}
	}
	return nil
}

func (c chatRenderer) renderRow(w io.Writer, row Row) error {
	user := row["user"]
	ts := row["time"]
	if c.timeFormat == "relative" {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			ts = reltime(t)
		}
	}
	msg := row["message"]
	if c.markdown {
		msg = renderMarkdown(msg, c.theme.GlamourStyle())
	}
	if _, err := fmt.Fprintf(w, "  %s · %s\n", user, ts); err != nil {
		return err
	}
	for _, line := range strings.Split(msg, "\n") {
		if _, err := fmt.Fprintf(w, "  %s\n", line); err != nil {
			return err
		}
	}
	meta := chatMeta(row)
	if meta == "" {
		_, err := fmt.Fprintln(w)
		return err
	}
	if c.color {
		meta = "\x1b[2m" + meta + ansiReset
	}
	if _, err := fmt.Fprintf(w, "  %s\n\n", meta); err != nil {
		return err
	}
	return nil
}

func chatMeta(row Row) string {
	fields := []string{"post_id", "root_id", "permalink", "files", "reactions"}
	var parts []string
	for _, f := range fields {
		v := row[f]
		if v == "" {
			continue
		}
		label := f
		if f == "root_id" {
			label = "thread"
		}
		parts = append(parts, fmt.Sprintf("%s=%s", label, v))
	}
	return strings.Join(parts, "  ")
}
