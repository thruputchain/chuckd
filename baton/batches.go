package baton

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) fetchBatches(
	ctx context.Context,
	headers []proto.BatchHeader,
) map[proto.BatchHash][]proto.ProposedTransaction {
	fetched := map[proto.BatchHash][]proto.ProposedTransaction{}
	for _, h := range headers {
		batch, ok := b.handler.LookupBatch(ctx, h.Batch_hash)
		if ok {
			fetched[h.Batch_hash] = batch
		}
	}
	return fetched
}
