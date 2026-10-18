package magnolia

import (
	"bytes"
	"context"
	"testing"

	"github.com/thruputchain/lib-go/librand"
)

func TestEncDec(t *testing.T) {
	ctx := context.Background()

	for _, v := range testEncDec {

		// gen random payload
		pay := make([]byte, v.NBytes)
		r := librand.NewRand()
		if _, err := r.Read(pay); err != nil {
			panic(err)
		}

		// encode
		enc := NewEncoder(ctx, v.Regime)
		cs := enc.Encode(pay)

		// decode
		dec := NewDecoder(ctx, v.Regime)
		ding := dec.StartDecoding(v.NBytes)

		// erase some sheaves
		for s := v.NumParitySheaves; s < v.NumPaySheaves+v.NumParitySheaves; s++ {
			for c := range cs {
				ding.Place(s, c, cs[c][s])
			}
		}

		repay, err := ding.Decode()
		if err != nil {
			t.Error(err)
		}

		if !bytes.Equal(pay, repay) {
			t.Errorf("corrupt reconstructed payload")
		}

	}
}

var testEncDec = []struct {
	NBytes int
	Regime
}{
	{NBytes: 11, Regime: MkRegime(2, 1, 4)},
	{NBytes: 100000, Regime: MkRegime(2, 1, 4)},
	{NBytes: 300e6, Regime: MkRegime(5, 2, 60e3)},
	{NBytes: 300e6, Regime: MkRegime(1, 2, 50e3)},
}
