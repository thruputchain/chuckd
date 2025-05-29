package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/baton/nominate"
	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type nominating struct {
	sdk.Tip
	context.CancelFunc
}

type nominated struct {
	*proto.Nomination
}

func canNominate(nominate *sdk.Tip, nominating *nominating, nominated *nominated) bool {
	if nominate == nil {
		return false
	}
	switch {
	case nominating == nil && nominated == nil:
		return true
	case nominating == nil && nominated != nil:
		return nominate.Height > nominated.Block_height
	case nominating != nil && nominated == nil:
		return nominate.Height > nominating.Height
	case nominating != nil && nominated != nil:
		return nominate.Height > nominating.Height && nominate.Height > nominated.Block_height
	}
	panic("unreachable")
}

func (b *Baton) nominate(ctx context.Context, tip sdk.Tip) (*proto.Nomination, proto.BlockID, error) {

	txs := []*proto.SignedTransaction{}
	for _, tx := range b.handler.ListTxs(ctx) {
		if !b.handler.IsCommittedTx(ctx, protoutil.HashSignedTx(tx)) {
			txs = append(txs, tx)
		}
	}

	scheduledTxs, rejectedTxs := b.handler.ScheduleTxs(ctx, txs)
	for _, rtx := range rejectedTxs {
		hash := protoutil.HashSignedTx(&rtx.Transaction)
		b.handler.AddCommittedTx(ctx, hash) // adding to committed so they are not tried again
		b.handler.RejectedTx(ctx, rtx)
	}
	if len(scheduledTxs) == 0 {
		return nil, 0, sdk.ErrNothingToNominate
	}

	// start working on proposing a block, while being ready to cancel if a block is externalized by someone else
	nom, err := nominate.PrepareNomination(ctx, b.handler, tip, scheduledTxs)
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
		proposedTxHashes = append(proposedTxHashes, protoutil.HashSignedTx(propsedTx.Tx))
		proposedTxNominations = append(proposedTxNominations,
			proto.NominatedTransaction{Nondeterministic_proposal_data: *propsedTx.Nondeterministic})
	}

	// construct block proposal
	proposal := &proto.Nomination{
		Block_height:            nom.BlockHeight,
		Block_hash:              nom.BlockHash,
		Prev_block_hash:         nom.PrevBlockHash,
		Transaction_hashes:      proposedTxHashes,
		Transaction_nominations: proposedTxNominations,
	}

	b.handler.Emit(ctx, sdk.EventNominating{Tip: tip, NumTx: len(proposedTxHashes), Nomination: proposal})

	// nominate to consensus
	prevSummary := proto.PrevProposalSummary{
		Block_hash: nom.PrevBlockHash,
	}
	// block until nomination is over
	b.handler.Nominate(ctx, nom.BlockHeight, proposal, &prevSummary) // ignore result
	b.handler.Emit(ctx, sdk.EventNominated{Tip: tip, NumTx: len(proposedTxHashes), Nomination: proposal})

	return proposal, nom.BlockID, nil
}
