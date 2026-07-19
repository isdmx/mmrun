package output

import (
	"sync"

	"github.com/charmbracelet/glamour"
)

var (
	mdCache   = map[string]*glamour.TermRenderer{}
	mdCacheMu sync.Mutex
)

func renderMarkdown(msg, glamourStyle string) string {
	if msg == "" {
		return ""
	}
	mdCacheMu.Lock()
	r, ok := mdCache[glamourStyle]
	if !ok {
		var err error
		r, err = glamour.NewTermRenderer(
			glamour.WithStandardStyle(glamourStyle),
			glamour.WithWordWrap(0),
		)
		if err != nil {
			mdCacheMu.Unlock()
			return msg
		}
		mdCache[glamourStyle] = r
	}
	mdCacheMu.Unlock()
	out, err := r.Render(msg)
	if err != nil {
		return msg
	}
	return out
}
