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

// New returns the Renderer for the requested mode against the given output file.
func New(requested string, out *os.File) Renderer {
	isTTY := IsTTY(out)
	mode := resolveMode(requested, isTTY)
	return rendererFor(mode, mode == "human" && isTTY)
}
