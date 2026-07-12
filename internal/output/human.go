package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type humanRenderer struct{ color bool }

const (
	ansiBold  = "\x1b[1m"
	ansiReset = "\x1b[0m"
)

func (h humanRenderer) Render(w io.Writer, r Result) error {
	if r.Text != "" {
		_, err := fmt.Fprintln(w, r.Text)
		return err
	}
	if r.Title != "" {
		title := r.Title
		if h.color {
			title = ansiBold + title + ansiReset
		}
		if _, err := fmt.Fprintln(w, title); err != nil {
			return err
		}
	}
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(r.Columns, "\t")); err != nil {
		return err
	}
	for _, row := range r.Rows {
		cells := make([]string, 0, len(r.Columns))
		for _, c := range r.Columns {
			cells = append(cells, row[c])
		}
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}
