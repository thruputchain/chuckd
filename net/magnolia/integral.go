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

type Place struct {
	NumBalls    int
	NumBins     int
	BallsPerBin int
}

func PlaceTight(nballs int, nbins int) Place {
	return Place{NumBalls: nballs, NumBins: nbins, BallsPerBin: divideRoundUp(nballs, nbins)}
}

func (x Place) Range(binIndex int) Range { // ball range
	return R(
		min(binIndex*x.BallsPerBin, x.NumBalls),
		min((binIndex+1)*x.BallsPerBin, x.NumBalls),
	)
}

type ShardChunk struct {
	NumBytes       int
	NumShards      int
	ChunksPerShard int
	BytesPerShard  int
	BytesPerChunk  int
}

func ShardChunkTight(nbytes, nshards, maxBytesPerChunk int) ShardChunk {

	bytesPerShard := divideRoundUp(nbytes, nshards)
	chunksPerShard := divideRoundUp(bytesPerShard, maxBytesPerChunk)
	bytesPerChunk := divideRoundUp(bytesPerShard, chunksPerShard)

	return ShardChunk{
		NumBytes:       nbytes,
		NumShards:      nshards,
		ChunksPerShard: chunksPerShard,
		BytesPerShard:  bytesPerShard,
		BytesPerChunk:  bytesPerChunk,
	}
}

func (x ShardChunk) Range(shardIndex int, chunkIndex int) Range { // byte range

	bytesToShards := Place{NumBalls: x.NumBytes, NumBins: x.NumShards, BallsPerBin: x.BytesPerShard}
	shardIntoBytes := bytesToShards.Range(shardIndex)

	shardBytesToChunks := Place{
		NumBalls:    shardIntoBytes.Len(),
		NumBins:     x.ChunksPerShard,
		BallsPerBin: x.BytesPerChunk,
	}
	chunkIntoShard := shardBytesToChunks.Range(chunkIndex)

	return shardIntoBytes.Subrange(chunkIntoShard)
}

func divideRoundUp(dividend, divisor int) int {
	return (dividend + divisor - 1) / divisor
}
