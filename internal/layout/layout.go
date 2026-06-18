// Package layout computes the box rectangle for every widget given the terminal
// size and the grid spec. All functions are pure.
package layout

import "github.com/locle97/lopanes/internal/config"

// Rect is a widget's box position and size in terminal cells.
type Rect struct {
	X, Y, W, H int
}

// Distribute splits total across len(weights) cells proportionally to their
// weights. Non-positive weights are treated as 1. Any rounding remainder is
// handed to the earliest cells so the result always sums to max(total, 0). The
// input slice is never mutated.
func Distribute(total int, weights []int) []int {
	n := len(weights)
	out := make([]int, n)
	if n == 0 {
		return out
	}
	if total < 0 {
		total = 0
	}

	norm := make([]int, n)
	sum := 0
	for i, w := range weights {
		if w <= 0 {
			w = 1
		}
		norm[i] = w
		sum += w
	}

	allocated := 0
	for i := 0; i < n; i++ {
		out[i] = total * norm[i] / sum
		allocated += out[i]
	}
	for i := 0; i < n && allocated < total; i++ {
		out[i]++
		allocated++
	}
	return out
}

// Compute returns the box rectangle for every widget for an interactive frame
// of termW x termH. Result is indexed [rowIndex][widgetIndex].
func Compute(termW, termH int, cfg config.Config) [][]Rect {
	heights := rowHeights(termH, cfg.Rows)
	return assemble(termW, heights, cfg.Rows)
}

// ComputePrint lays the grid out for print mode at a fixed width. Fixed rows
// keep their line count; weighted rows grow to fit the tallest widget content
// in the row (content lines + 2 border lines, minimum 3). contentHeights is
// indexed [rowIndex][widgetIndex].
func ComputePrint(width int, cfg config.Config, contentHeights [][]int) [][]Rect {
	heights := make([]int, len(cfg.Rows))
	for ri, row := range cfg.Rows {
		if row.HeightFixed > 0 {
			heights[ri] = row.HeightFixed
			continue
		}
		maxContent := 0
		for wi := range row.Widgets {
			ch := 0
			if ri < len(contentHeights) && wi < len(contentHeights[ri]) {
				ch = contentHeights[ri][wi]
			}
			if ch > maxContent {
				maxContent = ch
			}
		}
		h := maxContent + 2 // top + bottom border
		if h < 3 {
			h = 3
		}
		heights[ri] = h
	}
	return assemble(width, heights, cfg.Rows)
}

// assemble fills rectangles given pre-computed per-row heights, distributing
// each row's width across its widgets by weight.
func assemble(width int, heights []int, rows []config.Row) [][]Rect {
	rects := make([][]Rect, len(rows))
	y := 0
	for ri, row := range rows {
		h := heights[ri]
		weights := make([]int, len(row.Widgets))
		for wi, w := range row.Widgets {
			weights[wi] = w.WidthWeight
		}
		widths := Distribute(width, weights)
		rects[ri] = make([]Rect, len(row.Widgets))
		x := 0
		for wi := range row.Widgets {
			rects[ri][wi] = Rect{X: x, Y: y, W: widths[wi], H: h}
			x += widths[wi]
		}
		y += h
	}
	return rects
}

// rowHeights assigns each row a line count: fixed rows take their count first,
// weighted rows split the remaining vertical space by weight (clamped to >= 0).
func rowHeights(termH int, rows []config.Row) []int {
	heights := make([]int, len(rows))
	fixedTotal := 0
	var weightedIdx []int
	var weights []int
	for i, r := range rows {
		if r.HeightFixed > 0 {
			heights[i] = r.HeightFixed
			fixedTotal += r.HeightFixed
		} else {
			weightedIdx = append(weightedIdx, i)
			weights = append(weights, r.HeightWeight)
		}
	}
	remaining := termH - fixedTotal
	if remaining < 0 {
		remaining = 0
	}
	dist := Distribute(remaining, weights)
	for k, idx := range weightedIdx {
		heights[idx] = dist[k]
	}
	return heights
}
