package layout

import (
	"testing"

	"github.com/locle97/lopanes/internal/config"
)

func cfgFromRows(rows ...config.Row) config.Config {
	return config.Config{Rows: rows}
}

func TestComputeWeightedRowsAndEqualWidths(t *testing.T) {
	cfg := cfgFromRows(
		config.Row{HeightWeight: 1, Widgets: []config.Widget{
			{WidthWeight: 1}, {WidthWeight: 1},
		}},
		config.Row{HeightWeight: 1, Widgets: []config.Widget{
			{WidthWeight: 1},
		}},
	)
	rects := Compute(20, 10, cfg)
	// Two equal-height rows of 5 each.
	if rects[0][0].H != 5 || rects[1][0].H != 5 {
		t.Fatalf("heights: %+v", rects)
	}
	if rects[0][0].Y != 0 || rects[1][0].Y != 5 {
		t.Fatalf("y offsets: %+v", rects)
	}
	// Row 0 widths split 20 -> 10 + 10, x offsets 0 and 10.
	if rects[0][0].W != 10 || rects[0][1].W != 10 {
		t.Fatalf("widths: %+v", rects[0])
	}
	if rects[0][0].X != 0 || rects[0][1].X != 10 {
		t.Fatalf("x offsets: %+v", rects[0])
	}
	if rects[1][0].W != 20 {
		t.Fatalf("single widget should span full width: %+v", rects[1][0])
	}
}

func TestComputeFixedAndWeightedMix(t *testing.T) {
	cfg := cfgFromRows(
		config.Row{HeightFixed: 3, Widgets: []config.Widget{{WidthWeight: 1}}},
		config.Row{HeightWeight: 1, Widgets: []config.Widget{{WidthWeight: 1}}},
		config.Row{HeightWeight: 1, Widgets: []config.Widget{{WidthWeight: 1}}},
	)
	rects := Compute(10, 13, cfg)
	// Fixed row takes 3; remaining 10 split across two weighted rows -> 5, 5.
	if rects[0][0].H != 3 {
		t.Fatalf("fixed row height = %d want 3", rects[0][0].H)
	}
	if rects[1][0].H != 5 || rects[2][0].H != 5 {
		t.Fatalf("weighted heights: %+v %+v", rects[1][0], rects[2][0])
	}
	if rects[0][0].Y != 0 || rects[1][0].Y != 3 || rects[2][0].Y != 8 {
		t.Fatalf("y offsets wrong: %d %d %d", rects[0][0].Y, rects[1][0].Y, rects[2][0].Y)
	}
}

func TestComputeWidthRemainderToEarliest(t *testing.T) {
	cfg := cfgFromRows(config.Row{HeightWeight: 1, Widgets: []config.Widget{
		{WidthWeight: 1}, {WidthWeight: 1}, {WidthWeight: 1},
	}})
	rects := Compute(10, 4, cfg)
	if rects[0][0].W != 4 || rects[0][1].W != 3 || rects[0][2].W != 3 {
		t.Fatalf("remainder placement: %d %d %d", rects[0][0].W, rects[0][1].W, rects[0][2].W)
	}
}

func TestComputeTinyTerminal(t *testing.T) {
	cfg := cfgFromRows(
		config.Row{HeightFixed: 10, Widgets: []config.Widget{{WidthWeight: 1}}},
		config.Row{HeightWeight: 1, Widgets: []config.Widget{{WidthWeight: 1}}},
	)
	rects := Compute(4, 5, cfg) // fixed row alone exceeds height
	if rects[1][0].H != 0 {
		t.Fatalf("weighted row should clamp to 0 when no space left, got %d", rects[1][0].H)
	}
}

func TestComputePrint(t *testing.T) {
	cfg := cfgFromRows(
		config.Row{HeightWeight: 1, Widgets: []config.Widget{{WidthWeight: 1}, {WidthWeight: 1}}},
		config.Row{HeightFixed: 6, Widgets: []config.Widget{{WidthWeight: 1}}},
	)
	contentHeights := [][]int{
		{3, 7}, // weighted row: max content 7 -> height 7+2 = 9
		{0},    // fixed row ignores content
	}
	rects := ComputePrint(20, cfg, contentHeights)
	if rects[0][0].H != 9 || rects[0][1].H != 9 {
		t.Fatalf("weighted print height = %d want 9", rects[0][0].H)
	}
	if rects[1][0].H != 6 {
		t.Fatalf("fixed print height = %d want 6", rects[1][0].H)
	}
	if rects[1][0].Y != 9 {
		t.Fatalf("second row y = %d want 9", rects[1][0].Y)
	}
	if rects[0][0].W != 10 || rects[0][1].W != 10 {
		t.Fatalf("print widths: %+v", rects[0])
	}
}
