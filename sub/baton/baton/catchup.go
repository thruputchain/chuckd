package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
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
	ns     protoutil.ProposalTxsSerialized
	txs    []proto.SignedTransaction
}

func verifiedCatchupArrival(
	source proto.ValidatorID,
	stamp time.Time,
	block *proto.CatchupBlock,
) (CatchupArrival, bool) {
	ns, txs, ok := protoutil.VerifyCatchupBlock(block)
	if !ok {
		return CatchupArrival{}, false
	}
	return CatchupArrival{
		source: source,
		stamp:  stamp,
		block:  block,
		ns:     ns,
		txs:    txs,
	}, true
}

func (a CatchupArrival) Proposer() proto.ValidatorID {
	return a.block.Consensus_basis.Value.Proposal_header.Proposer
}

func (a CatchupArrival) Provider() proto.ValidatorID {
	return a.source
}

func (a CatchupArrival) Block() *proto.CatchupBlock {
	return a.block
}

func (a CatchupArrival) BlockHeight() proto.BlockHeight {
	return a.block.Consensus_basis.Value.Proposal_header.Block_height
}

func (a CatchupArrival) ConsensusBasis() *proto.ConsensusBasis {
	return &a.block.Consensus_basis
}

func (a CatchupArrival) ConsensusValue() *proto.ConsensusValue {
	return &a.block.Consensus_basis.Value
}

func (a CatchupArrival) ValueHash() proto.ConsensusValueHash {
	return protoutil.HashValueToProto(a.ConsensusValue())
}

func (a CatchupArrival) PrevValueHash() proto.ConsensusValueHash {
	return a.block.Consensus_basis.Value.Prev_hash
}

func (a CatchupArrival) SignedTransactions() []proto.SignedTransaction {
	return a.txs
}

func (a CatchupArrival) ProposalTxsSerialized() protoutil.ProposalTxsSerialized {
	return a.ns
}

type catchupRequest struct {
	spans []Span
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

func (b *Baton) attemptCatchup(ctx context.Context, spans []Span) {

	id := b.handler.GenerateID64(ctx)

	// mark pending request
	b.catchupWaiting[id] = catchupRequest{
		spans: spans,
	}

	// XXX: move this into a catchup agent
	// XXX: long catchup requests should be split into multiple requests
	go func(id batonkit.ID64, spans []Span) {

		var result catchupArrivals

		defer func() {
			b.lockNoDrive()
			// unmark pending request
			delete(b.catchupWaiting, id)
			// update known catchup blocks
			b.catchupArrivals = append(b.catchupArrivals, result...)
			if result.IsEmpty() {
				// Let the heartbeat schedule the next attempt. Driving immediately
				// here creates a tight retry loop when no providers are known.
				b.unlockNoDrive()
				return
			}
			b.unlock()
		}()

		// find sources
		srcs := b.handler.GetCatchupSources(ctx, int64(id))
		if len(srcs) == 0 {
			liblog.Errorf(ctx, "no catchup sources found among known validators")
			return
		}
		src := srcs[0]

		c, err := b.handler.DialCatchup(ctx, src)
		if err != nil {
			liblog.Errorf(ctx, "catchup dial: %v", err)
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
			liblog.Errorf(ctx, "catchup retrieval: %v", err)
			return
		}

		// verify the signature of the source
		if reply.Signer.Public_key.Scheme != proto.Ed25519 || src.Public_key.Scheme != proto.Ed25519 {
			liblog.Infof(ctx, "dropping catchup result: unsupported public key scheme (signer=%d, source=%d)",
				reply.Signer.Public_key.Scheme, src.Public_key.Scheme)
			return
		}
		if !protoutil.VerifyXdr(b.catchupResultM, src.Public_key, &reply.Result, reply.Signature) {
			liblog.Errorf(ctx, "catchup with invalid signature")
			return
		}

		now := b.handler.Now()
		for _, block := range reply.Result.Blocks {
			if arrival, ok := verifiedCatchupArrival(protoutil.PublicVID(src), now, &block); ok {
				result = append(result, arrival)
			} else {
				liblog.Infof(ctx, "invalid catchup block")
			}
		}

	}(id, spans)
}
