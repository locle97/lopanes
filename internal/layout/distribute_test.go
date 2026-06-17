package layout

import (
	"reflect"
	"testing"
)

func TestDistribute(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		weights []int
		want    []int
	}{
		{"equal exact", 10, []int{1, 1}, []int{5, 5}},
		{"remainder to earliest", 10, []int{1, 1, 1}, []int{4, 3, 3}},
		{"weighted", 10, []int{1, 4}, []int{2, 8}},
		{"weighted remainder", 11, []int{1, 1}, []int{6, 5}},
		{"zero total", 0, []int{1, 2}, []int{0, 0}},
		{"negative total clamps", -4, []int{1, 1}, []int{0, 0}},
		{"single", 7, []int{3}, []int{7}},
		{"zero weight treated as one", 4, []int{0, 0}, []int{2, 2}},
		{"empty", 5, []int{}, []int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Distribute(tt.total, tt.weights)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Distribute(%d,%v) = %v want %v", tt.total, tt.weights, got, tt.want)
			}
			sum := 0
			for _, v := range got {
				sum += v
			}
			wantSum := tt.total
			if wantSum < 0 {
				wantSum = 0
			}
			if len(tt.weights) > 0 && sum != wantSum {
				t.Fatalf("sum = %d want %d (must fill total exactly)", sum, wantSum)
			}
		})
	}
}

func TestDistributeDoesNotMutateInput(t *testing.T) {
	weights := []int{0, 0}
	Distribute(4, weights)
	if weights[0] != 0 || weights[1] != 0 {
		t.Fatalf("input mutated: %v", weights)
	}
}
