package magnolia

import (
	"context"
	"fmt"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/lib/libstat"
)

func TestBenchEncDec(t *testing.T) {
	ctx := context.Background()
	ed := newBenchEncDec(ctx, 600e3, 5, 2, 64e3)
	ctr := libstat.Counter{}
	for i := 0; i < 100; i++ {
		ed.Run(&ctr)
	}
	fmt.Printf("%s/sec\n", humanize.Bytes(uint64(ctr.SpeedPerSec())))
}

type benchEncDec struct {
	nPaySheaves    int
	nParitySheaves int
	pay            []byte
	enc            *Encoder
	dec            *Decoder
}

func newBenchEncDec(
	ctx context.Context,
	nPayBytes,
	nPaySheaves,
	nParitySheaves,
	maxBytesPerChunk int,
) *benchEncDec {

	pay := make([]byte, nPayBytes)
	r := librand.New()
	if _, err := r.Read(pay); err != nil {
		panic(err)
	}

	return &benchEncDec{
		nPaySheaves:    nPaySheaves,
		nParitySheaves: nParitySheaves,
		pay:            pay,
		enc:            NewEncoder(ctx, nPaySheaves, nParitySheaves, maxBytesPerChunk),
		dec:            NewDecoder(ctx, nPaySheaves, nParitySheaves, maxBytesPerChunk),
	}
}

func (x *benchEncDec) Run(ctr *libstat.Counter) {

	cs := x.enc.Encode(x.pay)
	ding := x.dec.StartDecoding(len(x.pay))

	// erase some sheaves
	for s := x.nParitySheaves; s < x.nPaySheaves+x.nParitySheaves; s++ {
		for c := range cs {
			ding.Place(s, c, cs[c][s])
		}
	}

	_, err := ding.Decode()
	if err != nil {
		panic("bug")
	}

	ctr.Add(float64(2 * len(x.pay)))
}
