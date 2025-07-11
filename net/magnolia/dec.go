package magnolia

import (
	"context"
	"fmt"

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

	cs := make(ChunkSheaf, place.ChunksPerSheaf)
	for c := range cs {
		cs[c] = make(Sheaf, x.numPaySheaves+x.numParitySheaves)
	}

	return &Decoding{dec: x, place: place, cs: cs}
}

type Decoding struct {
	dec   *Decoder
	place BytesToSheafChunks
	cs    ChunkSheaf
}

// XXX: (sheafIndex, chunkIndex) can be packed into symbolIndex
func (x *Decoding) Place(sheafIndex int, chunkIndex int, data Block) error {
	if len(data) != x.place.BytesPerChunk {
		return fmt.Errorf("invalid chunk size")
	}
	x.cs[chunkIndex][sheafIndex] = data
	return nil
}

func (x *Decoding) Decode() ([]byte, error) {

	for _, sheaf := range x.cs {
		if err := x.dec.dec.ReconstructData(sheaf); err != nil {
			return nil, err
		}
	}

	pay := make([]byte, x.place.NumBytes)
	for c := range x.cs {
		for s := range x.cs[c] {
			r := x.place.Range(s, c)
			copy(pay[r.From:r.To], x.cs[c][s][:r.Len()])
		}
	}

	return pay, nil
}
