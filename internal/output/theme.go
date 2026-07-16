package output

import _ "github.com/alecthomas/chroma/v2" // syntax highlighting for code blocks

// Theme defines ANSI styling for each visual element in human-mode output.
// A nil/zero-value Theme means "no styling".
type Theme struct {
	UserColor       string
	TimeColor       string
	ChannelColor    string
	SystemStyle     string
	BotStyle        string
	CodeLangStyle   string // chroma style name for code blocks w/ language
	CodeNoLangStyle string
	InlineCodeStyle string
	LinkStyle       string
	MentionStyle    string
	TreeMarker      string
}

// IsNone reports whether the theme deliberately suppresses all styling.
func (t Theme) IsNone() bool { return t.UserColor == "" && t.TimeColor == "" }

// DarkTheme is the default theme for dark terminal backgrounds.
var DarkTheme = Theme{
	UserColor:       "\x1b[38;5;%dm",
	TimeColor:       "\x1b[38;5;242m",
	ChannelColor:    "\x1b[38;5;72m",
	SystemStyle:     "\x1b[3;38;5;242m",
	BotStyle:        "\x1b[3;38;5;127m",
	CodeLangStyle:   "monokai",
	CodeNoLangStyle: "\x1b[1;48;5;236m",
	InlineCodeStyle: "\x1b[48;5;236m",
	LinkStyle:       "\x1b[4;38;5;69m",
	MentionStyle:    "\x1b[1;38;5;220m",
	TreeMarker:      "\x1b[1m",
}

// LightTheme is optimised for light/white terminal backgrounds.
var LightTheme = Theme{
	UserColor:       "\x1b[38;5;%dm",
	TimeColor:       "\x1b[38;5;238m",
	ChannelColor:    "\x1b[38;5;25m",
	SystemStyle:     "\x1b[3;38;5;242m",
	BotStyle:        "\x1b[3;38;5;54m",
	CodeLangStyle:   "github",
	CodeNoLangStyle: "\x1b[1;48;5;254m",
	InlineCodeStyle: "\x1b[48;5;254m",
	LinkStyle:       "\x1b[4;38;5;33m",
	MentionStyle:    "\x1b[1;38;5;172m",
	TreeMarker:      "\x1b[1m",
}

// MinimalTheme uses bold/italic only — no ANSI colors.
var MinimalTheme = Theme{
	SystemStyle:     "\x1b[3m",
	BotStyle:        "\x1b[3m",
	CodeLangStyle:   "bw",
	CodeNoLangStyle: "\x1b[1m",
	LinkStyle:       "\x1b[4m",
	MentionStyle:    "\x1b[1m",
	TreeMarker:      "\x1b[1m",
}

// resolveTheme picks the active theme from color and theme preferences.
func resolveTheme(_color, theme string) Theme {
	switch theme {
	case "light":
		return LightTheme
	case "minimal":
		return MinimalTheme
	default:
		return DarkTheme
	}
}
