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
	BallsToBins
	Bins []Range
}{
	{MkBallsToBins(0, 2), []Range{{0, 0}, {0, 0}}},
	{MkBallsToBins(1, 3), []Range{{0, 1}, {1, 1}, {1, 1}}},
	{MkBallsToBins(2, 1), []Range{{0, 2}}},
	{MkBallsToBins(3, 2), []Range{{0, 2}, {2, 3}}},
	{MkBallsToBins(4, 2), []Range{{0, 2}, {2, 4}}},
	{MkBallsToBins(5, 2), []Range{{0, 3}, {3, 5}}},
	{MkBallsToBins(5, 3), []Range{{0, 2}, {2, 4}, {4, 5}}},
}
