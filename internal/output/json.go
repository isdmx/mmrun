package output

import (
	"encoding/json"
	"io"
)

type jsonRenderer struct{}

func (jsonRenderer) Render(w io.Writer, r Result) error {
	payload := map[string]any{}
	if r.Text != "" {
		payload["text"] = r.Text
	}
	if r.Title != "" {
		payload["title"] = r.Title
	}
	if r.Rows != nil {
		rows := make([]Row, 0, len(r.Rows))
		for _, row := range r.Rows {
			if row["_type"] == "thread_header" {
				continue
			}
			rows = append(rows, row)
		}
		payload["rows"] = rows
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
