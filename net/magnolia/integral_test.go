package magnolia

import (
	"testing"
)

func TestPour(t *testing.T) {
	for i, v := range testPour {
		for bin := range v.Bins {
			got := v.Range(bin)
			exp := v.Bins[bin]
			if got != exp {
				t.Errorf("test %d, bin %d, got %v, expecting %v", i, bin, got, exp)
			}
		}
	}
}

var testPour = []struct {
	Place
	Bins []Range
}{
	{PlaceTight(0, 2), []Range{{0, 0}, {0, 0}}},
	{PlaceTight(1, 3), []Range{{0, 1}, {1, 1}, {1, 1}}},
	{PlaceTight(2, 1), []Range{{0, 2}}},
	{PlaceTight(3, 2), []Range{{0, 2}, {2, 3}}},
	{PlaceTight(4, 2), []Range{{0, 2}, {2, 4}}},
	{PlaceTight(5, 2), []Range{{0, 3}, {3, 5}}},
	{PlaceTight(5, 3), []Range{{0, 2}, {2, 4}, {4, 5}}},
}
