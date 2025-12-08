package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type catchupArrivals []CatchupArrival

func (a catchupArrivals) IsEmpty() bool {
	return len(a) == 0
}

func (a catchupArrivals) Find(h proto.BlockHeight) (CatchupArrival, bool) {
	for _, cb := range a {
		if cb.BlockHeight() == h {
			return cb, true
		}
	}
	return CatchupArrival{}, false
}

type CatchupArrival struct {
	source proto.ValidatorID
	stamp  time.Time
	block  *proto.CatchupBlock
	cvhash proto.ConsensusValueHash
}

func (a CatchupArrival) Provider() proto.ValidatorID {
	return a.source
}

func (a CatchupArrival) Block() *proto.CatchupBlock {
	return a.block
}

func (a CatchupArrival) BlockHeight() proto.BlockHeight {
	return a.block.Consensus_value.Nomination.Block_height
}

func (a CatchupArrival) BlockHash() proto.BlockHash {
	return a.block.Consensus_value.Nomination.Block_hash
}

func (a CatchupArrival) PrevBlockHash() proto.BlockHash {
	return a.block.Consensus_value.Nomination.Prev_block_hash
}

func (a CatchupArrival) ConsensusValue() *proto.ConsensusValue {
	return &a.block.Consensus_value
}

func (a CatchupArrival) ConsensusValueHash() proto.ConsensusValueHash {
	return a.cvhash
}

type catchupRequest struct {
	validator proto.ValidatorID
	spans     []Span
}

// baton

func (b *Baton) findCatchupBlockAtHeight(h proto.BlockHeight) (CatchupArrival, bool) {
	return b.catchupArrivals.Find(h)
}

func (b *Baton) findCatchupRequestsAtHeight(h proto.BlockHeight) []batonkit.ID64 {
	var reqs []batonkit.ID64
	for id, q := range b.catchupWaiting {
		for _, span := range q.spans {
			if span.From <= h && h < span.To {
				reqs = append(reqs, id)
			}
		}
	}
	return reqs
}

func (b *Baton) purgeCatchup(_ context.Context) {
	b.catchupArrivals = nil
}

func (b *Baton) requestCatchup(ctx context.Context, spans []Span) {

	id := b.handler.GenerateID64(ctx)

	src := b.handler.GetCatchupSources(ctx, int64(id))
	must.Assertf(ctx, len(src) > 0, "no catchup sources")

	// mark pending request
	b.catchupWaiting[id] = catchupRequest{
		validator: protoutil.PublicVID(src[0]),
		spans:     spans,
	}

	// XXX: move this into a catchup agent
	go func(src proto.ValidatorPublicInfo, id batonkit.ID64, spans []Span) {

		var result catchupArrivals

		defer func() {
			b.lock()
			defer b.unlock()

			// unmark pending request
			delete(b.catchupWaiting, id)
			// update known catchup blocks
			b.catchupArrivals = append(b.catchupArrivals, result...)
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
		if !protoutil.VerifyXdr(b.catchupResultM, src.Public_key, &reply.Result, reply.Signature) {
			liblog.Errorf("catchup with invalid signature")
			return
		}

		now := b.handler.Now()
		for _, block := range reply.Result.Blocks {
			if protoutil.VerifyCatchupBlock(b.txM, &block) {
				result = append(result,
					CatchupArrival{
						source: protoutil.PublicVID(src),
						stamp:  now,
						block:  &block,
						cvhash: b.hashCV(&block.Consensus_value),
					})
			}
		}

	}(src[0], id, spans)
}
