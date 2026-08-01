package progress

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestTransport_ByteCounting(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, 200) // 100 upload + 100 download
	b.width = 80

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		w.Header().Set("Content-Length", "100")
		w.Write(bytes.Repeat([]byte("x"), 100))
	}))
	defer server.Close()

	tr := NewTransport(http.DefaultTransport, b)
	client := &http.Client{Transport: tr}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(strings.Repeat("x", 100)))
	req.ContentLength = 100
	resp, _ := client.Do(req)
	io.ReadAll(resp.Body)
	resp.Body.Close()

	out := buf.String()
	if !strings.Contains(out, "100%") {
		t.Errorf("should show 100%% after full transfer, got %q", out)
	}
}

func TestReader_Progress(t *testing.T) {
	var buf bytes.Buffer
	b := NewBar(&buf, 100)
	b.width = 80
	r := NewReader(strings.NewReader(strings.Repeat("x", 100)), b)
	data := make([]byte, 50)
	n, _ := r.Read(data)
	if n != 50 {
		t.Fatalf("read %d, want 50", n)
	}
	out := buf.String()
	if !strings.Contains(out, "50%") {
		t.Errorf("after reading 50 of 100, should show 50%%, got %q", out)
	}
}
