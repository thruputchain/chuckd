package magnolia

import (
	"testing"

	"github.com/groundhogchain/chuckd/lib/form"
)

func TestBallsIntoBins(t *testing.T) {
	for i, v := range testBallsIntoBins {
		for bin := range v.Bins {
			got := v.Range(bin)
			exp := v.Bins[bin]
			if got != exp {
				t.Errorf("test %d, bin %d, got %v, expecting %v", i, bin, got, exp)
			}
		}
	}
}

var testBallsIntoBins = []struct {
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

func TestPlace(t *testing.T) {
	for i, v := range testPlace {

		got := MkBytesToSheafChunks(v.NBytes, v.NSheaves, v.NBytesPerChunk)
		if got != v.Place {
			t.Errorf("test %d, expecting %v, got %v", i, form.Display(v.Place), form.Display(got))
		}

		for s := 0; s < v.Place.NumSheaves; s++ {
			for c := 0; c < v.Place.ChunksPerSheaf; c++ {
				got := got.Range(s, c)
				exp := v.R[s][c]
				if got != exp {
					t.Errorf("test %d, expecting %v, got %v", i, form.Display(exp), form.Display(got))
				}
			}
		}
	}
}

var testPlace = []struct {
	NBytes         int
	NSheaves       int
	NBytesPerChunk int
	Place          BytesToSheafChunks
	R              [][]Range // sheaf, chunk
}{
	{
		NBytes: 11, NSheaves: 2,
		NBytesPerChunk: 4,
		Place: BytesToSheafChunks{
			NumBytes:       11,
			NumSheaves:     2,
			ChunksPerSheaf: 2,
			BytesPerSheaf:  6,
			BytesPerChunk:  3,
		},
		R: [][]Range{
			{{0, 3}, {3, 6}},  // sheaf 0, bytes [0:6]
			{{6, 9}, {9, 11}}, // sheaf 1, bytes [6:11]
		},
	},
}
