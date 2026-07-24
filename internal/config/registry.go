package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type setting struct {
	description string
	def         string
	raw         bool
	validate    func(string) error
	get         func(*Config) string
	set         func(*Config, string) error
}

func enumValidator(allowed ...string) func(string) error {
	return func(v string) error {
		if v == "" {
			return nil
		}
		for _, a := range allowed {
			if v == a {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %s", strings.Join(allowed, ", "))
	}
}

func posIntValidator(v string) error {
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("must be an integer")
	}
	if n < 1 {
		return fmt.Errorf("must be >= 1")
	}
	return nil
}

func setInt(target *int) func(*Config, string) error {
	return func(_ *Config, v string) error {
		if v == "" {
			*target = 0
			return nil
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("must be an integer")
		}
		*target = n
		return nil
	}
}

// settingsFor returns the settings bound to a specific Config pointer.
//
//nolint:funlen // declarative settings table
func settingsFor(c *Config) map[string]setting {
	return map[string]setting{
		"server_url": {
			description: "Mattermost server URL", def: "",
			validate: func(string) error { return nil },
			get:      func(c *Config) string { return c.ServerURL },
			set:      func(c *Config, v string) error { c.ServerURL = v; return nil },
		},
		"default_team": {
			description: "team used for bare channel names", def: "",
			validate: func(string) error { return nil },
			get:      func(c *Config) string { return c.DefaultTeam },
			set:      func(c *Config, v string) error { c.DefaultTeam = v; return nil },
		},
		"output_mode": {
			description: "output mode: auto|human|ai|json", def: "auto",
			validate: enumValidator("auto", "human", "ai", "json"),
			get:      func(c *Config) string { return c.OutputMode },
			set:      func(c *Config, v string) error { c.OutputMode = v; return nil },
		},
		"default_limit": {
			description: "default message page size", def: "50",
			raw:      true,
			validate: posIntValidator,
			get:      func(c *Config) string { return strconv.Itoa(c.DefaultLimit()) },
			set:      setInt(&c.DefaultLimit_),
		},
		"preview_len": {
			description: "message preview length (runes)", def: "140",
			raw:      true,
			validate: posIntValidator,
			get:      func(c *Config) string { return strconv.Itoa(c.PreviewLen()) },
			set:      setInt(&c.PreviewLen_),
		},
		"color": {
			description: "colorized output: auto|always|never", def: "auto",
			validate: enumValidator("auto", "always", "never"),
			get:      func(c *Config) string { return c.Color() },
			set:      func(c *Config, v string) error { c.ColorMode = v; return nil },
		},
		"download_dir": {
			description: "download directory (blank = XDG default)", def: "",
			validate: func(string) error { return nil },
			get:      func(c *Config) string { return c.DownloadDir() },
			set:      func(c *Config, v string) error { c.DownloadDir_ = v; return nil },
		},
		"columns": {
			description: "default columns for read/search (e.g. -permalink)", def: "",
			validate: validateColumnsSpec,
			get:      func(c *Config) string { return c.Columns },
			set:      func(c *Config, v string) error { c.Columns = v; return nil },
		},
		"format": {
			description: "output format for read/search/thread/mentions: table|tree",
			def:         "table",
			validate:    enumValidator("table", "tree"),
			get:         func(c *Config) string { return c.Format() },
			set:         func(c *Config, v string) error { c.Format_ = v; return nil },
		},
		"theme": {
			description: "color theme: dark|light|minimal",
			def:         "dark",
			validate:    enumValidator("dark", "light", "minimal"),
			get:         func(c *Config) string { return c.Theme() },
			set:         func(c *Config, v string) error { c.Theme_ = v; return nil },
		},
		"markdown": {
			description: "render markdown in messages: true|false",
			def:         "true",
			validate:    enumValidator("true", "false"),
			get:         func(c *Config) string { return c.Markdown_ },
			set:         func(c *Config, v string) error { c.Markdown_ = v; return nil },
		},

		"style": {
			description: "output style: table|chat|tree",
			def:         "table",
			validate:    enumValidator("table", "chat", "tree"),
			get:         func(c *Config) string { return c.Style() },
			set:         func(c *Config, v string) error { c.Style_ = v; return nil },
		},
		"time_format": {
			description: "timestamp format: rfc3339|relative",
			def:         "rfc3339",
			validate:    enumValidator("rfc3339", "relative"),
			get:         func(c *Config) string { return c.TimeFormat() },
			set:         func(c *Config, v string) error { c.TimeFormat_ = v; return nil },
		},
		"full": {
			description: "always show full message text",
			def:         "false",
			validate:    enumValidator("true", "false"),
			get: func(c *Config) string {
				if c.Full() {
					return "true"
				}
				return "false"
			},
			set: func(c *Config, v string) error { c.Full_ = v; return nil },
		},
		"threads_only": {
			description: "show only root posts (no replies) by default",
			def:         "false",
			validate:    enumValidator("true", "false"),
			get: func(c *Config) string {
				if c.ThreadsOnly() {
					return "true"
				}
				return "false"
			},
			set: func(c *Config, v string) error { c.ThreadsOnly_ = v; return nil },
		},
		"auto_mark_read": {
			description: "automatically mark channels/threads as read on view",
			def:         "false",
			validate:    enumValidator("true", "false"),
			get: func(c *Config) string {
				if c.AutoMarkRead() {
					return "true"
				}
				return "false"
			},
			set: func(c *Config, v string) error { c.AutoMarkRead_ = v; return nil },
		},
	}
}

// MessageColumns lists the column names valid for read/search output. It lives
// here so config can validate the columns preference without importing cmd.
var MessageColumns = []string{"time", "channel", "user", "files", "root_id", "post_id", "permalink", "message"}

func validateColumnsSpec(v string) error {
	if v == "" {
		return nil
	}
	valid := map[string]bool{}
	for _, c := range MessageColumns {
		valid[c] = true
	}
	for _, tok := range strings.Split(v, ",") {
		tok = strings.TrimSpace(tok)
		name := strings.TrimLeft(tok, "+-")
		if name == "" || !valid[name] {
			return fmt.Errorf("unknown column %q (valid: %s)", name, strings.Join(MessageColumns, ", "))
		}
	}
	return nil
}

// Keys returns all known setting keys, sorted.
func Keys() []string {
	m := settingsFor(&Config{})
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Get returns the effective value of a key.
func Get(c *Config, key string) (string, error) {
	s, ok := settingsFor(c)[key]
	if !ok {
		return "", fmt.Errorf("unknown key %q (valid: %s)", key, strings.Join(Keys(), ", "))
	}
	return s.get(c), nil
}

// Set validates and applies a key's value on the Config.
func Set(c *Config, key, value string) error {
	s, ok := settingsFor(c)[key]
	if !ok {
		return fmt.Errorf("unknown key %q (valid: %s)", key, strings.Join(Keys(), ", "))
	}
	if err := s.validate(value); err != nil {
		return fmt.Errorf("invalid value for %q: %w", key, err)
	}
	return s.set(c, value)
}

// Describe returns the description and default for a key.
func Describe(key string) (desc, def string, ok bool) {
	s, found := settingsFor(&Config{})[key]
	if !found {
		return "", "", false
	}
	return s.description, s.def, true
}

// Template renders a fully-commented config.toml with default values.
func Template() string {
	settings := settingsFor(&Config{})
	var b strings.Builder
	b.WriteString("# mmrun configuration\n")
	b.WriteString("# Generated defaults; edit as needed. See 'mmrun config list'.\n\n")
	for _, key := range Keys() {
		s := settings[key]
		fmt.Fprintf(&b, "# %s\n", s.description)
		if s.raw {
			fmt.Fprintf(&b, "%s = %s\n\n", key, s.def)
		} else {
			fmt.Fprintf(&b, "%s = %q\n\n", key, s.def)
		}
	}
	return b.String()
}
