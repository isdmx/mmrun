// Package output renders command results as human-readable text, AI-friendly
// plain text, or JSON.
package output

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Row is a single record of key/value fields.
type Row map[string]string

// Result is the typed, renderer-agnostic output every command produces.
type Result struct {
	Title   string
	Columns []string
	Rows    []Row
	// Text is used for freeform single-value output (e.g. a posted message id).
	Text string
}

// Renderer writes a Result in a specific format.
type Renderer interface {
	Render(w io.Writer, r Result) error
}

// Options tune rendering (colorization and highlighting) for human output.
type Options struct {
	Color      string   // "auto" | "always" | "never" | "" (=auto)
	Highlight  []string // terms to emphasize in cells (human mode only)
	Format     string   // "table" (default) or "tree"
	Theme      string   // "dark"|"light"|"minimal"|""
	Style      string   // "table"|"chat"|"tree" (default "table")
	TimeFormat string   // "rfc3339"|"relative" (default "rfc3339")
}

// colorEnabled reports whether ANSI color should be emitted for the given color
// mode and TTY state. The NO_COLOR env var forces it off.
func colorEnabled(mode string, isTTY bool) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // "auto" / ""
		return isTTY
	}
}

// IsTTY reports whether the file is an interactive terminal.
func IsTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// resolveMode maps the requested mode + TTY state to a concrete mode.
func resolveMode(requested string, isTTY bool) string {
	if requested == "auto" || requested == "" {
		if isTTY {
			return "human"
		}
		return "ai"
	}
	return requested
}

func rendererFor(mode string, color bool) Renderer {
	switch mode {
	case "json":
		return jsonRenderer{}
	case "human":
		return humanRenderer{color: color}
	default: // "ai"
		return aiRenderer{}
	}
}

// New returns the Renderer for the requested mode against the given output file,
// with default (auto) options.
func New(requested string, out *os.File) Renderer {
	return NewWithOptions(requested, out, Options{})
}

// NewWithOptions returns a Renderer honoring the given color/highlight options.
func NewWithOptions(requested string, out *os.File, opts Options) Renderer {
	isTTY := IsTTY(out)
	mode := resolveMode(requested, isTTY)
	if mode != "human" {
		return rendererFor(mode, false)
	}
	if opts.Format == "tree" || opts.Style == "tree" {
		themeObj := resolveTheme(opts.Color, opts.Theme)
		col := colorEnabled(opts.Color, isTTY)
		if themeObj.IsNone() {
			col = false
		}
		return treeBlockRenderer{color: col, theme: themeObj, timeFormat: opts.TimeFormat}
	}
	themeObj := resolveTheme(opts.Color, opts.Theme)
	if themeObj.IsNone() {
		return humanRenderer{color: false, highlight: opts.Highlight, timeFormat: opts.TimeFormat}
	}
	colorMode := opts.Color
	if opts.Theme != "" && (colorMode == "" || colorMode == "auto") {
		colorMode = "always"
	}
	col := colorEnabled(colorMode, isTTY)
	switch opts.Style {
	case "chat":
		return chatRenderer{color: col, theme: themeObj, timeFormat: opts.TimeFormat}
	case "tree":
		return treeBlockRenderer{color: col, theme: themeObj, timeFormat: opts.TimeFormat}
	default:
		return humanRenderer{color: col, highlight: opts.Highlight, theme: themeObj, timeFormat: opts.TimeFormat}
	}
}
