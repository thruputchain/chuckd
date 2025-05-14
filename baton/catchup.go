package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type catchupBlocks []catchupBlock

func (bs catchupBlocks) IsEmpty() bool {
	return len(bs) == 0
}

func (bs catchupBlocks) Find(h proto.BlockHeight) (catchupBlock, bool) {
	for _, cb := range bs {
		if cb.BlockHeight() == h {
			return cb, true
		}
	}
	return catchupBlock{}, false
}

type catchupBlock struct {
	source proto.ValidatorID
	stamp  time.Time
	block  *proto.CatchupBlock
	txs    []*proto.ProposedTransaction
}

func (b catchupBlock) Source() proto.ValidatorID {
	return b.source
}

func (b catchupBlock) BlockHeight() proto.BlockHeight {
	return b.block.Block_height
}

func (b catchupBlock) BlockHash() proto.BlockHash {
	return b.block.Block_hash
}

func (b catchupBlock) PrevBlockHash() proto.BlockHash {
	return b.block.Prev_block_hash
}

type catchupRequest struct {
	validator proto.ValidatorID
	spans     []Span
}

// baton

func (b *Baton) findCatchupBlockAtHeight(h proto.BlockHeight) (catchupBlock, bool) {
	return b.catchupBlocks.Find(h)
}

func (b *Baton) findCatchupRequestsAtHeight(h proto.BlockHeight) []ID64 {
	var reqs []ID64
	for id, q := range b.catchupRequests {
		for _, span := range q.spans {
			if span.From <= h && h < span.To {
				reqs = append(reqs, id)
			}
		}
	}
	return reqs
}

func (b *Baton) purgeCatchup(_ context.Context) {
	b.catchupBlocks = nil
}

func (b *Baton) requestCatchup(ctx context.Context, spans []Span) {

	src := b.handler.GetCatchupSources(ctx)
	must.Assertf(ctx, len(src) > 0, "no catchup sources")

	// mark pending request
	id := b.handler.GenerateID64(ctx)
	b.catchupRequests[id] = catchupRequest{
		validator: protoutil.ValidatorIDFromPublic(src[0]),
		spans:     spans,
	}

	go func(src proto.ValidatorPublicInfo, id ID64, spans []Span) {

		var result catchupBlocks

		defer func() {
			b.synchronize <- func() {
				// unmark pending request
				delete(b.catchupRequests, id)
				// update known catchup blocks
				b.catchupBlocks = append(b.catchupBlocks, result...)
			}
		}()

		c, err := b.handler.DialCatchup(ctx, src)
		if err != nil {
			log.Errorf("catchup dial: %v", err)
			return
		}
		defer c.Close()

		req := proto.CatchupBlocksRequest{
			Timestamp: b.handler.Now().UnixNano(),
			Spans:     make([]proto.CatchupSpan, len(spans)),
		}
		for i, span := range spans {
			req.Spans[i].From_block_height = span.From
			req.Spans[i].To_block_height = span.To
		}
		signed := proto.SignedCatchupBlocksRequest{
			Signer:    b.handler.Signer(),
			Signature: b.handler.SignXdr(&req),
			Request:   req,
		}

		reply, err := c.Catchup_blocks(ctx, signed)
		if err != nil {
			log.Errorf("catchup retrieval: %v", err)
			return
		}

		if !b.handler.VerifyXdr(src.Public_key, &reply.Result, reply.Signature) {
			log.Errorf("catchup with invalid signature")
			return
		}

		now := b.handler.Now()
		for _, block := range reply.Result.Blocks {
			result = append(result, catchupBlock{
				source: protoutil.ValidatorIDFromPublic(src),
				stamp:  now,
				block:  &block,
				txs:    protoutil.RepackageProposedTransactions(block.Proposed_transactions),
			})
		}

	}(src[0], id, spans)
}
