// Package layout computes the box rectangle for every widget given the terminal
// size and the grid spec. All functions are pure.
package layout

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
