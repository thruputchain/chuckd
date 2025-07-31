package baton

import (
	"context"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/baton/nominate"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type nominating struct {
	batonapi.Tip
	context.CancelFunc
}

type nominated struct {
	time.Time
	*proto.Nomination
}

func (b *Baton) canNominate(nominate *batonapi.Tip, nominating *nominating, nominated *nominated) bool {
	if nominate == nil {
		return false
	}
	switch {
	case nominating == nil && nominated == nil:
		return true
	case nominating == nil && nominated != nil:
		return nominate.Height > nominated.Block_height ||
			b.handler.Now().Sub(nominated.Time) > b.nominationTTL
	case nominating != nil && nominated == nil:
		return nominate.Height > nominating.Height
	case nominating != nil && nominated != nil:
		return nominate.Height > nominating.Height && (nominate.Height > nominated.Block_height ||
			b.handler.Now().Sub(nominated.Time) > b.nominationTTL)
	}
	panic("unreachable")
}

func (b *Baton) nominate(ctx context.Context, tip batonapi.Tip) (*proto.Nomination, proto.BlockID, error) {

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
	nom, err := nominate.PrepareNomination(ctx, b.handler, tip, scheduledTxs, b.callsPerExec, b.batchSize)
	if err != nil {
		// XXX: ctx may be cancelled
		// XXX: ErrNoExecs
		return nil, 0, err
	}

	proposedTxHashes := []proto.SignedTransactionHash{}
	proposedTxNominations := []proto.NominatedTransaction{}
	for _, propsedTx := range nom.ProposedTxs {
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
		Block_height:            nom.BlockHeight,
		Block_hash:              nom.BlockHash,
		Prev_block_hash:         nom.PrevBlockHash,
		Transaction_hashes:      proposedTxHashes,
		Transaction_nominations: proposedTxNominations,
	}

	b.handler.Emit(ctx, batonapi.EventNominating{Tip: tip, NumTx: len(proposedTxHashes), Nomination: nomination})

	// nominate to consensus
	prevSummary := proto.PrevNominationSummary{
		Block_hash: nom.PrevBlockHash,
	}
	// block until nomination is over
	b.handler.Nominate(ctx, nom.BlockHeight, nomination, &prevSummary) // ignore result
	b.handler.Emit(ctx, batonapi.EventNominated{Tip: tip, NumTx: len(proposedTxHashes), Nomination: nomination})

	return nomination, nom.BlockID, nil
}
