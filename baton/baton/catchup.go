package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type catchupBlockShells []CatchupBlockShell

func (bs catchupBlockShells) IsEmpty() bool {
	return len(bs) == 0
}

func (bs catchupBlockShells) Find(h proto.BlockHeight) (CatchupBlockShell, bool) {
	for _, cb := range bs {
		if cb.BlockHeight() == h {
			return cb, true
		}
	}
	return CatchupBlockShell{}, false
}

type CatchupBlockShell struct {
	Source proto.ValidatorID
	Stamp  time.Time
	Block  *proto.CatchupBlock
	CVHash proto.ConsensusValueHash
}

func (b CatchupBlockShell) Provider() proto.ValidatorID {
	return b.Source
}

func (b CatchupBlockShell) BlockHeight() proto.BlockHeight {
	return b.Block.Consensus_value.Nomination.Block_height
}

func (b CatchupBlockShell) BlockHash() proto.BlockHash {
	return b.Block.Consensus_value.Nomination.Block_hash
}

func (b CatchupBlockShell) PrevBlockHash() proto.BlockHash {
	return b.Block.Consensus_value.Nomination.Prev_block_hash
}

func (b CatchupBlockShell) ConsensusValueHash() proto.ConsensusValueHash {
	return b.CVHash
}

type catchupRequest struct {
	validator proto.ValidatorID
	spans     []Span
}

// baton

func (b *Baton) findCatchupBlockAtHeight(h proto.BlockHeight) (CatchupBlockShell, bool) {
	return b.catchupBlocks.Find(h)
}

func (b *Baton) findCatchupRequestsAtHeight(h proto.BlockHeight) []batonkit.ID64 {
	var reqs []batonkit.ID64
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

	go func(src proto.ValidatorPublicInfo, id batonkit.ID64, spans []Span) {

		var result catchupBlockShells

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
			Timestamp: protoutil.TimepointFromTime(b.handler.Now()),
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

		// verify the signature of the source
		if !b.handler.VerifyXdr(b.catchupResultM, src.Public_key, &reply.Result, reply.Signature) {
			liblog.Errorf("catchup with invalid signature")
			return
		}

		now := b.handler.Now()
		for _, block := range reply.Result.Blocks {
			if protoutil.VerifyCatchupBlock(b.txM, &block) {
				result = append(result,
					CatchupBlockShell{
						Source: protoutil.PublicVID(src),
						Stamp:  now,
						Block:  &block,
						CVHash: b.hashCV(&block.Consensus_value),
					})
			}
		}

	}(src[0], id, spans)
}
