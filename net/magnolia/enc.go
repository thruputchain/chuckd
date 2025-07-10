package magnolia

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/klauspost/reedsolomon"
)

type Encoder struct {
	numPaySheaves    int
	numParitySheaves int
	maxBytesPerChunk int
	enc              reedsolomon.Encoder
}

func NewEncoder(
	ctx context.Context,
	numPaySheaves int,
	numParitySheaves int,
	maxBytesPerChunk int,
) *Encoder {
	enc, err := reedsolomon.New(
		int(numPaySheaves),
		int(numParitySheaves),
	)
	must.NoError(ctx, err)
	return &Encoder{
		numPaySheaves:    numPaySheaves,
		numParitySheaves: numParitySheaves,
		maxBytesPerChunk: maxBytesPerChunk,
		enc:              enc,
	}
}

func (x *Encoder) Encode(pay []byte) ChunkSheaf {

	place := MkBytesToSheafChunks(len(pay), x.numPaySheaves, x.maxBytesPerChunk)

	cs := make(ChunkSheaf, place.ChunksPerSheaf)
	for c := range cs {
		cs[c] = make(Sheaf, x.numPaySheaves+x.numParitySheaves)
		// use pay slice to fill
		for s := 0; s < x.numPaySheaves; s++ {
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
		for s := x.numPaySheaves; s < x.numPaySheaves+x.numParitySheaves; s++ {
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
