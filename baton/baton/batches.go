package baton

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) fetchBatches(
	ctx context.Context,
	hashes []proto.BatchHash,
) map[proto.BatchHash][]*proto.ProposedTransaction {
	fetched := map[proto.BatchHash][]*proto.ProposedTransaction{}
	for _, h := range hashes {
		batch, ok := b.handler.LookupBatch(ctx, h)
		if ok {
			fetched[h] = batch
		}
	}
	return fetched
}
