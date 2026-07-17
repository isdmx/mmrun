package output

import (
	"testing"
	"time"
)

func TestReltime(t *testing.T) {
	now := time.Now()
	if got := reltime(now.Add(-30 * time.Second)); got != "just now" {
		t.Errorf("30s = %q", got)
	}
	if got := reltime(now.Add(-5 * time.Minute)); got != "5m ago" {
		t.Errorf("5m = %q", got)
	}
	if got := reltime(now.Add(-3 * time.Hour)); got != "3h ago" {
		t.Errorf("3h = %q", got)
	}
	if got := reltime(now.Add(-48 * time.Hour)); got != "2d ago" {
		t.Errorf("2d = %q", got)
	}
	old := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if got := reltime(old); got != "Jan 2" {
		t.Errorf("old = %q, want Jan 2", got)
	}
}
