package output

import (
	"fmt"
	"io"
	"strings"
)

type aiRenderer struct{}

func (aiRenderer) Render(w io.Writer, r Result) error {
	if r.Text != "" {
		_, err := fmt.Fprintln(w, r.Text)
		return err
	}
	if r.Title != "" {
		if _, err := fmt.Fprintf(w, "# %s\n", r.Title); err != nil {
			return err
		}
	}
	for _, row := range r.Rows {
		parts := make([]string, 0, len(r.Columns))
		for _, c := range r.Columns {
			parts = append(parts, fmt.Sprintf("%s=%s", c, row[c]))
		}
		if _, err := fmt.Fprintln(w, strings.Join(parts, "\t")); err != nil {
			return err
		}
	}
	return nil
}
