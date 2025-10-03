package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type catchupBlocks []CatchupBlock

func (bs catchupBlocks) IsEmpty() bool {
	return len(bs) == 0
}

func (bs catchupBlocks) Find(h proto.BlockHeight) (CatchupBlock, bool) {
	for _, cb := range bs {
		if cb.BlockHeight() == h {
			return cb, true
		}
	}
	return CatchupBlock{}, false
}

type CatchupBlock struct {
	Source proto.ValidatorID
	Stamp  time.Time
	Block  *proto.CatchupBlock
	Txs    []*proto.ProposedTransaction
}

func (b CatchupBlock) Provider() proto.ValidatorID {
	return b.Source
}

func (b CatchupBlock) BlockHeight() proto.BlockHeight {
	return b.Block.Block_height
}

func (b CatchupBlock) BlockHash() proto.BlockHash {
	return b.Block.Block_hash
}

func (b CatchupBlock) PrevBlockHash() proto.BlockHash {
	return b.Block.Prev_block_hash
}

type catchupRequest struct {
	validator proto.ValidatorID
	spans     []Span
}

// baton

func (b *Baton) findCatchupBlockAtHeight(h proto.BlockHeight) (CatchupBlock, bool) {
	return b.catchupBlocks.Find(h)
}

func (b *Baton) findCatchupRequestsAtHeight(h proto.BlockHeight) []batonapi.ID64 {
	var reqs []batonapi.ID64
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

	id := b.handler.GenerateID64(ctx)

	src := b.handler.GetCatchupSources(ctx, int64(id))
	must.Assertf(ctx, len(src) > 0, "no catchup sources")

	// mark pending request
	b.catchupRequests[id] = catchupRequest{
		validator: protoutil.PublicVID(src[0]),
		spans:     spans,
	}

	go func(src proto.ValidatorPublicInfo, id batonapi.ID64, spans []Span) {

		var result catchupBlocks

		defer func() {
			b.lock()
			defer b.unlock()

			// unmark pending request
			delete(b.catchupRequests, id)
			// update known catchup blocks
			b.catchupBlocks = append(b.catchupBlocks, result...)
		}()

		c, err := b.handler.DialCatchup(ctx, src)
		if err != nil {
			liblog.Errorf("catchup dial: %v", err)
			return
		}
		defer c.Close()

		req := proto.CatchupBlocksRequest{
			Timestamp: uint64(b.handler.Now().UnixNano()),
			Spans:     make([]proto.CatchupSpan, len(spans)),
		}
		for i, span := range spans {
			req.Spans[i].From_block_height = span.From
			req.Spans[i].To_block_height = span.To
		}
		signed := proto.SignedCatchupBlocksRequest{
			Signer:    b.handler.Signer(),
			Signature: b.handler.SignXdr(b.catchupReqM, &req),
			Request:   req,
		}

		reply, err := c.Catchup_solicit(ctx, signed)
		if err != nil {
			liblog.Errorf("catchup retrieval: %v", err)
			return
		}

		if !b.handler.VerifyXdr(b.catchupResultM, src.Public_key, &reply.Result, reply.Signature) {
			liblog.Errorf("catchup with invalid signature")
			return
		}

		now := b.handler.Now()
		for _, block := range reply.Result.Blocks {
			result = append(result, CatchupBlock{
				Source: protoutil.PublicVID(src),
				Stamp:  now,
				Block:  &block,
				Txs:    protoutil.UnpackProposedTransactions(block.Proposed_transactions),
			})
		}

	}(src[0], id, spans)
}
