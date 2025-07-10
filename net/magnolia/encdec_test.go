package magnolia

import (
	"bytes"
	"context"
	"testing"

	"github.com/groundhogchain/chuckd/lib/librand"
)

func TestEncDec(t *testing.T) {
	ctx := context.Background()

	for _, v := range testEncDec {

		// gen random payload
		pay := make([]byte, v.NBytes)
		r := librand.New()
		if _, err := r.Read(pay); err != nil {
			panic(err)
		}

		// encode
		enc := NewEncoder(ctx, v.NPaySheaves, v.NParitySheaves, v.MaxBytesPerChunk)
		cs := enc.Encode(pay)

		// decode
		dec := NewDecoder(ctx, v.NPaySheaves, v.NParitySheaves, v.MaxBytesPerChunk)
		ing := dec.StartDecoding(v.NBytes)

		// erase some sheaves
		for s := v.NParitySheaves; s < v.NPaySheaves+v.NParitySheaves; s++ {
			for c := range cs {
				ing.Place(s, c, cs[c][s])
			}
		}

		repay, err := ing.Decode()
		if err != nil {
			t.Error(err)
		}

		if bytes.Compare(pay, repay) != 0 {
			t.Errorf("corrupt reconstructed payload")
		}

	}
}

var testEncDec = []struct {
	NBytes           int
	NPaySheaves      int
	NParitySheaves   int
	MaxBytesPerChunk int
}{
	{NBytes: 11, NPaySheaves: 2, NParitySheaves: 1, MaxBytesPerChunk: 4},
}
