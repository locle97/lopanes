// Package widget renders a single widget result into a bordered box, clipping
// and padding the body to a rectangle while preserving ANSI escape sequences.
package widget

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"

	"github.com/locle97/quash-board/internal/layout"
	"github.com/locle97/quash-board/internal/runner"
)

// State is the rendering state of a widget.
type State int

const (
	StatePending State = iota // no result yet
	StateOK                   // last run succeeded
	StateError                // last run failed or timed out
)

// View is everything needed to render one box.
type View struct {
	Title    string
	Body     string // current good stdout (or last good output when in error)
	State    State
	ErrLabel string // e.g. "exit 1" or "timed out"
	ErrTail  string // stderr to show under the error indicator
}

// Theme controls styling so tests can render deterministically.
type Theme struct {
	Dim func(string) string
}

// DefaultTheme dims text with a faint SGR attribute.
func DefaultTheme() Theme {
	faint := lipgloss.NewStyle().Faint(true)
	return Theme{Dim: func(s string) string { return faint.Render(s) }}
}

// PlainTheme applies no styling (used by --no-color and tests).
func PlainTheme() Theme {
	return Theme{Dim: func(s string) string { return s }}
}

// FromResult maps a runner.Result and prior good output into a View, returning
// the View and the new "last good output" to remember. On failure the prior
// good output is preserved so the box keeps showing it (dimmed).
func FromResult(title, lastGood string, res runner.Result) (View, string) {
	v := View{Title: title}
	if res.Err == nil && !res.TimedOut && res.ExitCode == 0 {
		v.State = StateOK
		v.Body = res.Stdout
		return v, res.Stdout
	}
	v.State = StateError
	switch {
	case res.TimedOut:
		v.ErrLabel = "timed out"
	case res.Err != nil:
		v.ErrLabel = "error"
	default:
		v.ErrLabel = fmt.Sprintf("exit %d", res.ExitCode)
	}
	v.ErrTail = res.Stderr
	v.Body = lastGood
	return v, lastGood
}

// ContentHeight returns how many body lines the View wants to display, used by
// print mode to size weighted rows.
func ContentHeight(v View) int {
	switch v.State {
	case StatePending:
		return 1
	case StateOK:
		n := len(splitLines(v.Body))
		if n < 1 {
			n = 1
		}
		return n
	default: // StateError
		n := 1 // indicator
		for _, ln := range splitLines(v.ErrTail) {
			if strings.TrimSpace(ln) != "" {
				n++
			}
		}
		if v.Body != "" {
			n += len(splitLines(v.Body))
		}
		return n
	}
}

// Render draws the View into a box of rect.W x rect.H. Every returned line is
// exactly rect.W cells wide and there are exactly rect.H lines.
func Render(v View, rect layout.Rect, theme Theme) string {
	w, h := rect.W, rect.H
	if w < 2 || h < 2 {
		return blankBlock(w, h)
	}
	innerW := w - 2
	innerH := h - 2

	top := topBorder(v.Title, w)
	bottom := "└" + strings.Repeat("─", innerW) + "┘"
	body := buildBody(v, innerW, innerH, theme)

	var b strings.Builder
	b.WriteString(top)
	for _, ln := range body {
		b.WriteByte('\n')
		b.WriteString("│")
		b.WriteString(ln)
		b.WriteString("│")
	}
	b.WriteByte('\n')
	b.WriteString(bottom)
	return b.String()
}

// topBorder renders "┌─ title ───...───┐" exactly w cells wide.
func topBorder(title string, w int) string {
	inner := w - 2
	maxLabel := inner - 4 // "─ " + " " + at least one trailing dash
	if maxLabel < 0 {
		return "┌" + strings.Repeat("─", inner) + "┐"
	}
	label := truncate.String(title, uint(maxLabel))
	used := 3 + visibleWidth(label) // "─ " (2) + label + " " (1)
	dashes := inner - used
	if dashes < 0 {
		dashes = 0
	}
	mid := "─ " + label + " " + strings.Repeat("─", dashes)
	return "┌" + mid + "┐"
}

// buildBody returns exactly innerH lines, each exactly innerW cells wide.
func buildBody(v View, innerW, innerH int, theme Theme) []string {
	var lines []string
	switch v.State {
	case StatePending:
		lines = []string{"…"}
	case StateOK:
		lines = splitLines(v.Body)
	default: // StateError — indicator first for visibility
		lines = append(lines, "⚠ "+v.ErrLabel)
		for _, ln := range splitLines(v.ErrTail) {
			if strings.TrimSpace(ln) != "" {
				lines = append(lines, "  "+ln)
			}
		}
		if v.Body != "" {
			for _, ln := range splitLines(v.Body) {
				lines = append(lines, theme.Dim(ln))
			}
		}
	}

	out := make([]string, innerH)
	for i := 0; i < innerH; i++ {
		if i < len(lines) {
			out[i] = fitLine(lines[i], innerW)
		} else {
			out[i] = strings.Repeat(" ", innerW)
		}
	}
	return out
}

// fitLine truncates s (ANSI-aware) to w cells, then right-pads to exactly w.
func fitLine(s string, w int) string {
	if w <= 0 {
		return ""
	}
	t := truncate.String(s, uint(w))
	if pad := w - visibleWidth(t); pad > 0 {
		t += strings.Repeat(" ", pad)
	}
	return t
}

// visibleWidth is the printed cell width ignoring ANSI escapes.
func visibleWidth(s string) int { return lipgloss.Width(s) }

func blankBlock(w, h int) string {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	line := strings.Repeat(" ", w)
	rows := make([]string, h)
	for i := range rows {
		rows[i] = line
	}
	return strings.Join(rows, "\n")
}

// splitLines splits on newlines, normalizing CRLF and dropping a single
// trailing empty line produced by a final newline.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines
}
