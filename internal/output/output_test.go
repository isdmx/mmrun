package output

import (
	"bytes"
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
	r := rendererFor("json", false)
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
	r := rendererFor("ai", false)
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
