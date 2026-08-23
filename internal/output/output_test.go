package output

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func sampleResult() Result {
	return Result{
		Title: "Teams",
		Rows: []Row{
			{"name": "eng", "display": "Engineering"},
			{"name": "ops", "display": "Operations"},
		},
		Columns: []string{"name", "display"},
	}
}

func TestJSONRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := rendererFor("json", false, "")
	if err := r.Render(&buf, sampleResult()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"name": "eng"`) || !strings.Contains(out, `"display": "Operations"`) {
		t.Errorf("json output missing fields:\n%s", out)
	}
}

func TestAIRenderer_NoANSI(t *testing.T) {
	var buf bytes.Buffer
	r := rendererFor("ai", false, "")
	if err := r.Render(&buf, sampleResult()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\x1b[") {
		t.Errorf("ai output must not contain ANSI escapes:\n%q", buf.String())
	}
	if !strings.Contains(buf.String(), "eng") {
		t.Errorf("ai output missing data")
	}
}

func TestModeSelection_AutoNonTTYIsAI(t *testing.T) {
	if got := resolveMode("auto", false); got != "ai" {
		t.Errorf("auto non-tty = %q, want ai", got)
	}
	if got := resolveMode("auto", true); got != "human" {
		t.Errorf("auto tty = %q, want human", got)
	}
	if got := resolveMode("json", true); got != "json" {
		t.Errorf("explicit json = %q, want json", got)
	}
}

func TestColorMode(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	if colorEnabled("never", true) {
		t.Error("never must disable color")
	}
	if !colorEnabled("always", false) {
		t.Error("always must enable color even without TTY")
	}
	if colorEnabled("auto", false) {
		t.Error("auto without TTY must be off")
	}
	if !colorEnabled("auto", true) {
		t.Error("auto with TTY must be on")
	}
}

func TestHumanHighlight(t *testing.T) {
	var buf bytes.Buffer
	r := humanRenderer{color: true, highlight: []string{"@alice"}, timeFormat: "rfc3339"}
	res := Result{
		Columns: []string{"user", "message"},
		Rows:    []Row{{"user": "@alice", "message": "hi @alice"}},
	}
	if err := r.Render(&buf, res); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Error("expected ANSI when highlighting with color")
	}
}

func TestNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if colorEnabled("always", true) {
		t.Error("NO_COLOR must force color off")
	}
}

func TestThemeColor_DarkProducesANSI(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	r := NewWithOptions("human", os.Stdout, Options{Theme: "dark"})
	hr, ok := r.(humanRenderer)
	if !ok || !hr.color {
		t.Error("dark theme should produce colored human renderer")
	}
}

func TestHumanCodeBlock(t *testing.T) {
	var buf bytes.Buffer
	r := humanRenderer{theme: DarkTheme, color: true, timeFormat: "rfc3339"}
	res := Result{
		Columns: []string{"user", "message"},
		Rows:    []Row{{"user": "alice", "message": "hey\n```python\nprint(1)\n```\ndone"}},
	}
	if err := r.Render(&buf, res); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "\x1b[") {
		t.Error("code block should contain ANSI from chroma")
	}
}

func TestQuietRenderer(t *testing.T) {
	var buf bytes.Buffer
	r := quietRenderer{column: "post_id"}
	res := Result{
		Columns: []string{"post_id", "message"},
		Rows: []Row{
			{"post_id": "abc123", "message": "hello"},
			{"post_id": "def456", "message": "world"},
		},
	}
	if err := r.Render(&buf, res); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 || lines[0] != "abc123" || lines[1] != "def456" {
		t.Errorf("expected abc123\\ndef456, got %q", buf.String())
	}
}

func TestQuietRendererSkipsEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := quietRenderer{column: "post_id"}
	res := Result{
		Columns: []string{"post_id", "message"},
		Rows: []Row{
			{"post_id": "", "message": "empty id"},
			{"post_id": "abc123", "message": "hello"},
		},
	}
	if err := r.Render(&buf, res); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "abc123\n" {
		t.Errorf("expected only abc123\\n, got %q", buf.String())
	}
}
