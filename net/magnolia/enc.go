package magnolia

import (
	"context"
	"math/rand"

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
	rand *rand.Rand,
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

type Chunk = []byte
type Sheaf = []Chunk

func (x *Encoder) Encode(pay []byte) []Sheaf {

	place := MkBytesToSheafChunks(len(pay), x.numPaySheaves, x.maxBytesPerChunk)

	sheaf := make([]Sheaf, x.numPaySheaves+x.numParitySheaves)
	for i := range sheaf {
		sheaf[i] = make([]Chunk, place.ChunksPerSheaf)
		// don't allocate chunks, will use pay slice
	}

	panic("XXX")
}
