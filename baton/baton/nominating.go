package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/nominate"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type nominating struct {
	OnTip  batonapi.Tip
	Cancel context.CancelFunc
}

type nominated struct {
	time.Time
	Nomination *proto.Nomination
}

func (b *Baton) canNominate(tipToNominateOn *batonapi.Tip, nominating *nominating, nominated *nominated) bool {
	if tipToNominateOn == nil {
		return false
	}
	switch {
	case nominating == nil && nominated == nil:
		return true
	case nominating == nil && nominated != nil:
		return tipToNominateOn.Height+1 > nominated.Nomination.Block_height ||
			b.handler.Now().Sub(nominated.Time) > b.nominationTTL
	case nominating != nil && nominated == nil:
		return tipToNominateOn.Height > nominating.OnTip.Height
	case nominating != nil && nominated != nil:
		return tipToNominateOn.Height > nominating.OnTip.Height &&
			(tipToNominateOn.Height+1 > nominated.Nomination.Block_height || b.handler.Now().Sub(nominated.Time) > b.nominationTTL)
	}
	panic("unreachable")
}

func (b *Baton) assembleNomination(ctx context.Context, tip batonapi.Tip) (*proto.Nomination, proto.BlockID, error) {

	txs := []*proto.SignedTransaction{}
	for _, tx := range b.handler.ListTxs(ctx) {
		if !b.handler.IsCommittedTx(ctx, protoutil.HashSignedTx(b.txM, tx)) {
			txs = append(txs, tx)
		}
	}

	scheduledTxs, rejectedTxs := b.handler.ScheduleTxs(ctx, txs)
	for _, rtx := range rejectedTxs {
		hash := protoutil.HashSignedTx(b.txM, &rtx.Transaction)
		b.handler.AddCommittedTx(ctx, hash) // adding to committed so they are not tried again
		b.handler.RejectedTx(ctx, rtx)
	}
	if len(scheduledTxs) == 0 {
		return nil, 0, batonapi.ErrNothingToNominate
	}

	// start working on proposing a block, while being ready to cancel if a block is externalized by someone else
	blockID := b.handler.GenerateBlockID(ctx)
	prepared, err := nominate.PrepareNomination(ctx, b.handler, tip, scheduledTxs, b.callsPerExec, b.batchSize, blockID)
	if err != nil {
		// XXX: ctx may be cancelled
		// XXX: ErrNoExecs
		return nil, 0, err
	}

	proposedTxHashes := []proto.SignedTransactionHash{}
	proposedTxNominations := []proto.NominatedTransaction{}
	for _, propsedTx := range prepared.ProposedTxs {
		if propsedTx.Err != nil {
			// XXX: should invalid transaction be marked AddCommittedTx so they are not tried again?
			continue
		}
		proposedTxHashes = append(proposedTxHashes, protoutil.HashSignedTx(b.txM, propsedTx.Tx))
		proposedTxNominations = append(proposedTxNominations,
			proto.NominatedTransaction{Nondeterministic_proposal_data: *propsedTx.Result})
	}

	// construct block nomination
	nomination := &proto.Nomination{
		Proposer:                b.handler.Signer(),
		Block_height:            prepared.BlockHeight,
		Block_hash:              prepared.BlockHash,
		Prev_block_hash:         prepared.PrevBlockHash,
		Transaction_hashes:      proposedTxHashes,
		Transaction_nominations: proposedTxNominations,
	}

	an := &assembledNomination{tip: tip, prepared: prepared, proto: nomination}
	_ = an

	b.handler.Emit(ctx, batonapi.EventNominating{Tip: tip, NumTx: len(proposedTxHashes), Nomination: nomination})

	// nominate to consensus
	prevSummary := proto.PrevNominationSummary{
		Block_hash: prepared.PrevBlockHash,
	}
	// block until nomination is over
	if ok := b.handler.Nominate(ctx, scpapi.Slot(prepared.BlockHeight), nomination, &prevSummary); ok {
		b.handler.Emit(ctx, batonapi.EventNominated{Tip: tip, NumTx: len(proposedTxHashes), Nomination: nomination})
	} else {
		liblog.Infof("nomination rejected on %v", tip.String())
	}

	return nomination, prepared.BlockID, nil
}

type assembledNomination struct {
	tip      batonapi.Tip
	prepared *nominate.Nomination
	proto    *proto.Nomination
}

func (b *Baton) announceNomination(ctx context.Context, an *assembledNomination) (*proto.Nomination, proto.BlockID, error) {
	b.handler.Emit(ctx, batonapi.EventNominating{Tip: an.tip, NumTx: len(an.proto.Transaction_hashes), Nomination: an.proto})

	// nominate to consensus
	prevSummary := proto.PrevNominationSummary{
		Block_hash: an.proto.Prev_block_hash,
	}
	// block until nomination is over
	if ok := b.handler.Nominate(ctx, scpapi.Slot(an.proto.Block_height), an.proto, &prevSummary); ok {
		b.handler.Emit(ctx, batonapi.EventNominated{Tip: an.tip, NumTx: len(an.proto.Transaction_hashes), Nomination: an.proto})
	} else {
		liblog.Infof("nomination rejected on %v", an.tip.String())
	}

	return an.proto, an.prepared.BlockID, nil
}
