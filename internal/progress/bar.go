// Package progress provides a terminal progress bar for file transfers.
package progress

import (
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/term"
)

// Bar writes a single-line progress bar to an io.Writer (typically os.Stderr).
type Bar struct {
	out       io.Writer
	total     int64
	current   int64
	label     string
	startTime time.Time
	width     int // terminal width, 0 if piped/unknown
}

// NewBar creates a progress bar writing to w. total is the expected byte count.
func NewBar(w io.Writer, total int64) *Bar {
	width := 80
	if f, ok := w.(interface{ Fd() uintptr }); ok {
		if tw, _, err := term.GetSize(int(f.Fd())); err == nil {
			width = tw
		}
	}
	return &Bar{out: w, total: total, startTime: time.Now(), width: width}
}

// SetLabel changes the label shown before the bar (e.g. "reading", "uploading").
func (b *Bar) SetLabel(s string) { b.label = s }

// Add advances the bar by n bytes and re-renders.
func (b *Bar) Add(n int64) {
	b.current += n
	b.render()
}

// Reset zeros the current counter while keeping total and label.
func (b *Bar) Reset() { b.current = 0 }

// Done renders the final 100% state and prints a newline.
func (b *Bar) Done() {
	b.current = b.total
	b.render()
	_, _ = fmt.Fprintln(b.out)
}

func (b *Bar) render() {
	pct := int64(0)
	if b.total > 0 {
		pct = b.current * 100 / b.total
	}
	elapsed := time.Since(b.startTime).Round(time.Second)
	sec := elapsed.Seconds()
	speed := float64(0)
	if sec > 0 {
		speed = float64(b.current) / sec
	}
	eta := "?"
	if speed > 0 && b.current > 0 {
		remaining := time.Duration(float64(b.total-b.current)/speed) * time.Second
		eta = remaining.Round(time.Second).String()
	}

	label := b.label
	if b.width < 40 {
		_, _ = fmt.Fprintf(b.out, "\r  %s... %d%%  %s / %s  %s/s",
			label, pct, humanSize(b.current), humanSize(b.total), humanSizeF(speed))
	} else {
		barWidth := b.width - 40
		if barWidth < 10 {
			barWidth = 10
		}
		var filled int
		if b.total > 0 {
			filled = int(int64(barWidth-1) * b.current / b.total)
			if filled >= barWidth-1 {
				filled = barWidth - 1
			}
		}
		bar := "[" + strings.Repeat("=", filled) + ">" + strings.Repeat(" ", barWidth-1-filled) + "]"
		_, _ = fmt.Fprintf(b.out, "\r  %-10s %s %3d%%  %s / %s  %s/s  ETA %s",
			label, bar, pct, humanSize(b.current), humanSize(b.total), humanSizeF(speed), eta)
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for r := n / unit; r >= unit; r /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func humanSizeF(n float64) string {
	if n <= 0 {
		return "0 B"
	}
	return humanSize(int64(n))
}
