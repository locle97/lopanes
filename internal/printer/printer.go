// Package printer renders the dashboard once, concurrently running every
// widget script, and returns the full frame as a string for stdout.
package printer

import (
	"context"
	"regexp"
	"sync"

	"github.com/charmbracelet/lipgloss"

	"github.com/locle97/lopanes/internal/config"
	"github.com/locle97/lopanes/internal/layout"
	"github.com/locle97/lopanes/internal/runner"
	"github.com/locle97/lopanes/internal/widget"
)

const defaultWidth = 80

// Options controls print rendering.
type Options struct {
	Width   int  // <= 0 means defaultWidth
	NoColor bool // strip ANSI from the final frame
}

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// Render runs every widget once and returns the rendered dashboard frame.
func Render(cfg config.Config, opts Options) string {
	width := opts.Width
	if width <= 0 {
		width = defaultWidth
	}

	// Widget widths are content-independent, so compute them first and use them
	// to inject WIDGET_W before running scripts concurrently.
	results := make([][]runner.Result, len(cfg.Rows))
	var wg sync.WaitGroup
	for ri, row := range cfg.Rows {
		results[ri] = make([]runner.Result, len(row.Widgets))
		weights := make([]int, len(row.Widgets))
		for wi, w := range row.Widgets {
			weights[wi] = w.WidthWeight
		}
		widths := layout.Distribute(width, weights)
		for wi, w := range row.Widgets {
			wg.Add(1)
			go func(ri, wi int, w config.Widget, boxW int) {
				defer wg.Done()
				env := runner.WidgetEnv(boxW-2, 0, width, 0)
				results[ri][wi] = runner.Run(context.Background(), runner.RunSpec{
					Script:  w.Script,
					Timeout: w.Timeout,
					Env:     env,
				})
			}(ri, wi, w, widths[wi])
		}
	}
	wg.Wait()

	// Build views and per-widget content heights.
	views := make([][]widget.View, len(cfg.Rows))
	contentHeights := make([][]int, len(cfg.Rows))
	for ri, row := range cfg.Rows {
		views[ri] = make([]widget.View, len(row.Widgets))
		contentHeights[ri] = make([]int, len(row.Widgets))
		for wi, w := range row.Widgets {
			v, _ := widget.FromResult(w.Title, w.Color, "", results[ri][wi])
			views[ri][wi] = v
			contentHeights[ri][wi] = widget.ContentHeight(v)
		}
	}

	rects := layout.ComputePrint(width, cfg, contentHeights)

	theme := widget.DefaultTheme()
	if opts.NoColor {
		theme = widget.PlainTheme()
	}
	rowStrs := make([]string, len(cfg.Rows))
	for ri := range views {
		cells := make([]string, len(views[ri]))
		for wi := range views[ri] {
			cells[wi] = widget.Render(views[ri][wi], rects[ri][wi], theme)
		}
		rowStrs[ri] = lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	}
	out := lipgloss.JoinVertical(lipgloss.Left, rowStrs...)
	if opts.NoColor {
		out = ansiRe.ReplaceAllString(out, "")
	}
	return out
}
