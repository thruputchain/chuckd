package magnolia

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/klauspost/reedsolomon"
)

type Regime struct {
	NumPaySheaves    int
	NumParitySheaves int
	MaxBytesPerChunk int
}

func MkRegime(numPaySheaves int, numParitySheaves int, maxBytesPerChunk int) Regime {
	return Regime{
		NumPaySheaves:    numPaySheaves,
		NumParitySheaves: numParitySheaves,
		MaxBytesPerChunk: maxBytesPerChunk,
	}
}

type Encoder struct {
	regime Regime
	enc    reedsolomon.Encoder
}

func NewEncoder(
	ctx context.Context,
	regime Regime,
) *Encoder {
	enc, err := reedsolomon.New(regime.NumPaySheaves, regime.NumParitySheaves)
	must.NoError(ctx, err)
	return &Encoder{
		regime: regime,
		enc:    enc,
	}
}

func (x *Encoder) Regime() Regime {
	return x.regime
}

func (x *Encoder) NumSheaves() int {
	return x.regime.NumPaySheaves + x.regime.NumParitySheaves
}

func (x *Encoder) NumPaySheaves() int {
	return x.regime.NumPaySheaves
}

func (x *Encoder) NumParitySheaves() int {
	return x.regime.NumParitySheaves
}

func (x *Encoder) MaxBytesPerChunk() int {
	return x.regime.MaxBytesPerChunk
}

func (x *Encoder) Encode(pay []byte) ChunkSheaf {

	place := MkBytesToSheafChunks(len(pay), x.regime.NumPaySheaves, x.regime.MaxBytesPerChunk)

	cs := make(ChunkSheaf, place.ChunksPerSheaf)
	for c := range cs {
		cs[c] = make(Sheaf, x.regime.NumPaySheaves+x.regime.NumParitySheaves)
		// use pay slice to fill
		for s := 0; s < x.regime.NumPaySheaves; s++ {
			r := place.Range(s, c)
			switch {
			case r.Len() == place.BytesPerChunk:
				cs[c][s] = pay[r.From:r.To]
			case r.Len() < place.BytesPerChunk: // padding
				b := make(Block, place.BytesPerChunk)
				copy(b[:r.Len()], pay[r.From:r.To])
				cs[c][s] = b
			default:
				panic("bug")
			}
		}
		for s := x.regime.NumPaySheaves; s < x.regime.NumPaySheaves+x.regime.NumParitySheaves; s++ {
			cs[c][s] = make(Block, place.BytesPerChunk) // XXX: can be optimized to one alloc
		}
		// encode parity sheaves
		err := x.enc.Encode(cs[c])
		if err != nil {
			panic("bug: " + err.Error())
		}
	}

	return cs
}
