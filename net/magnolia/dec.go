package magnolia

import (
	"context"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/klauspost/reedsolomon"
)

type Decoder struct {
	numPaySheaves    int
	numParitySheaves int
	maxBytesPerChunk int
	dec              reedsolomon.Encoder
}

func NewDecoder(
	ctx context.Context,
	rand *rand.Rand,
	numPaySheaves int,
	numParitySheaves int,
	maxBytesPerChunk int,
) *Decoder {
	dec, err := reedsolomon.New(
		int(numPaySheaves),
		int(numParitySheaves),
	)
	must.NoError(ctx, err)
	return &Decoder{
		numPaySheaves:    numPaySheaves,
		numParitySheaves: numParitySheaves,
		maxBytesPerChunk: maxBytesPerChunk,
		dec:              dec,
	}
}

func (x *Decoder) StartDecoding(numPayBytes int) *Decoding {

	place := MkBytesToSheafChunks(numPayBytes, x.numPaySheaves, x.maxBytesPerChunk)

	sheaf := make([]Sheaf, x.numPaySheaves+x.numParitySheaves)
	for i := range sheaf {
		sheaf[i] = make([]Chunk, place.ChunksPerSheaf)
		// don't allocate chunks
		// XXX: check how RS indicates missing chunks
	}

	return &Decoding{
		dec:   x,
		place: place,
		sheaf: sheaf,
	}
}

type Decoding struct {
	dec   *Decoder
	place BytesToSheafChunks
	sheaf []Sheaf
}

// XXX: (sheafIndex, chunkIndex) can be packed into symbolIndex
func (x *Decoding) Place(sheafIndex int, chunkIndex int, data Chunk) error {
	panic("XXX")
}

func (x *Decoding) IsDecodable() bool {
	panic("XXX")
}

func (x *Decoding) Decode() []byte {
	panic("XXX")
}
