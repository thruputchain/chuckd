package magnolia

type Range struct {
	From int
	To   int // not inclusive
}

func R(from int, to int) Range {
	return Range{From: from, To: to}
}

func (x Range) Subrange(sub Range) Range {
	return Range{
		From: x.From + sub.From,
		To:   x.From + sub.To,
	}
}

func (x Range) Len() int {
	return x.To - x.From
}

type BallsToBins struct {
	NumBalls    int
	NumBins     int
	BallsPerBin int
}

func MkBallsToBins(nballs int, nbins int) BallsToBins {
	return BallsToBins{NumBalls: nballs, NumBins: nbins, BallsPerBin: divideRoundUp(nballs, nbins)}
}

func (x BallsToBins) Range(binIndex int) Range { // ball range
	return R(
		min(binIndex*x.BallsPerBin, x.NumBalls),
		min((binIndex+1)*x.BallsPerBin, x.NumBalls),
	)
}

type BytesToSheafChunks struct {
	NumBytes       int
	NumSheaves     int
	ChunksPerSheaf int
	BytesPerSheaf  int
	BytesPerChunk  int
}

func MkBytesToSheafChunks(nbytes, nsheaves, maxBytesPerChunk int) BytesToSheafChunks {

	bytesPerSheaf := divideRoundUp(nbytes, nsheaves)
	chunksPerSheaf := divideRoundUp(bytesPerSheaf, maxBytesPerChunk)
	bytesPerChunk := divideRoundUp(bytesPerSheaf, chunksPerSheaf)

	return BytesToSheafChunks{
		NumBytes:       nbytes,
		NumSheaves:     nsheaves,
		ChunksPerSheaf: chunksPerSheaf,
		BytesPerSheaf:  bytesPerSheaf,
		BytesPerChunk:  bytesPerChunk,
	}
}

func (x BytesToSheafChunks) Range(sheafIndex int, chunkIndex int) Range { // byte range

	bytesToSheaves := BallsToBins{NumBalls: x.NumBytes, NumBins: x.NumSheaves, BallsPerBin: x.BytesPerSheaf}
	sheafIntoBytes := bytesToSheaves.Range(sheafIndex)

	sheafBytesToChunks := BallsToBins{
		NumBalls:    sheafIntoBytes.Len(),
		NumBins:     x.ChunksPerSheaf,
		BallsPerBin: x.BytesPerChunk,
	}
	chunkIntoSheaf := sheafBytesToChunks.Range(chunkIndex)

	return sheafIntoBytes.Subrange(chunkIntoSheaf)
}

func divideRoundUp(dividend, divisor int) int {
	return (dividend + divisor - 1) / divisor
}
