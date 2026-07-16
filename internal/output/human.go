package output

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

type humanRenderer struct {
	color     bool
	highlight []string
	theme     Theme
}

const (
	ansiBold      = "\x1b[1m"
	ansiHighlight = "\x1b[33m" // yellow
	ansiReset     = "\x1b[0m"
)

func (h humanRenderer) emphasize(s string) string {
	if !h.color || len(h.highlight) == 0 {
		return s
	}
	for _, term := range h.highlight {
		if term == "" {
			continue
		}
		s = strings.ReplaceAll(s, term, ansiHighlight+term+ansiReset)
	}
	return s
}

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
			cells = append(cells, h.emphasize(h.styleCell(c, row[c])))
		}
		if _, err := fmt.Fprintln(tw, strings.Join(cells, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func (h humanRenderer) styleCell(col, val string) string {
	if !h.color || h.theme.IsNone() {
		return val
	}
	switch col {
	case "time":
		return h.theme.TimeColor + val + ansiReset
	case "channel":
		return h.theme.ChannelColor + val + ansiReset
	case "message":
		return highlightCodeBlocks(val, h.theme)
	}
	return val
}

var codeBlockRe = regexp.MustCompile("(?s)" + "```" + `(\w*)\n(.+?)` + "```")

func highlightCodeBlocks(msg string, th Theme) string {
	return codeBlockRe.ReplaceAllStringFunc(msg, func(block string) string {
		matches := codeBlockRe.FindStringSubmatch(block)
		if len(matches) < 3 {
			return th.CodeNoLangStyle + block + ansiReset
		}
		lang := matches[1]
		code := matches[2]
		if lang == "" {
			if th.CodeNoLangStyle != "" {
				return th.CodeNoLangStyle + code + ansiReset
			}
			return code
		}
		lexer := lexers.Get(lang)
		if lexer == nil {
			lexer = lexers.Fallback
		}
		style := styles.Get(th.CodeLangStyle)
		if style == nil {
			style = styles.Fallback
		}
		iterator, err := lexer.Tokenise(nil, code)
		if err != nil {
			return th.CodeNoLangStyle + code + ansiReset
		}
		var buf bytes.Buffer
		_ = formatters.TTY.Format(&buf, style, iterator)
		return buf.String()
	})
}
