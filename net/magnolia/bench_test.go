package magnolia

import (
	"context"
	"fmt"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/groundhogchain/lib-go/librand"
	"github.com/groundhogchain/lib-go/libstat"
)

func TestBenchEncDec(t *testing.T) {
	ctx := context.Background()
	ed := newBenchEncDec(ctx, 600e3, MkRegime(5, 2, 64e3))
	ctr := libstat.Counter{}
	for i := 0; i < 100; i++ {
		ed.Run(&ctr)
	}
	fmt.Printf("%s/sec\n", humanize.Bytes(uint64(ctr.SpeedPerSec())))
}

type benchEncDec struct {
	regime Regime
	pay    []byte
	enc    *Encoder
	dec    *Decoder
}

func newBenchEncDec(
	ctx context.Context,
	nPayBytes int,
	regime Regime,
) *benchEncDec {

	pay := make([]byte, nPayBytes)
	r := librand.NewRand()
	if _, err := r.Read(pay); err != nil {
		panic(err)
	}

	return &benchEncDec{
		regime: regime,
		pay:    pay,
		enc:    NewEncoder(ctx, regime),
		dec:    NewDecoder(ctx, regime),
	}
}

func (x *benchEncDec) Run(ctr *libstat.Counter) {

	cs := x.enc.Encode(x.pay)
	ding := x.dec.StartDecoding(len(x.pay))

	// erase some sheaves
	for s := x.regime.NumParitySheaves; s < x.regime.NumPaySheaves+x.regime.NumParitySheaves; s++ {
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
