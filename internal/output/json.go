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
		payload["rows"] = r.Rows
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
