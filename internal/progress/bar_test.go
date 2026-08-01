package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestBar_Render(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, 100)
	b.SetLabel("test")
	b.width = 80
	b.Add(50)
	out := buf.String()
	if !strings.Contains(out, "test") {
		t.Errorf("missing label in %q", out)
	}
	if !strings.Contains(out, "50%") {
		t.Errorf("missing percentage in %q", out)
	}
	if !strings.Contains(out, "50 B / 100 B") {
		t.Errorf("missing sizes in %q", out)
	}
}

func TestBar_Done(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, 100)
	b.SetLabel("test")
	b.width = 80
	b.Done()
	out := buf.String()
	if !strings.Contains(out, "100%") {
		t.Errorf("done should show 100%%, got %q", out)
	}
	if !strings.Contains(out, "100 B / 100 B") {
		t.Errorf("done should show full size, got %q", out)
	}
}

func TestBar_Reset(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, 200)
	b.width = 80
	b.Add(100)
	b.Reset()
	if b.current != 0 {
		t.Errorf("Reset should zero current, got %d", b.current)
	}
}

func TestBar_NarrowTerminal(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, 1024)
	b.SetLabel("uploading")
	b.width = 0
	b.Add(512)
	out := buf.String()
	if strings.Contains(out, "[") {
		t.Errorf("narrow terminal should not show bar graphic, got %q", out)
	}
	if !strings.Contains(out, "50%") {
		t.Errorf("missing percentage in %q", out)
	}
}
