package baton

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/nominate"
	"github.com/groundhogchain/chuckd/baton/sdk"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

var ErrNothingToNominate = fmt.Errorf("nothing to nominate")

type nominating struct {
	sdk.Tip
	context.CancelFunc
}

type nominated struct {
	*proto.Proposal
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

func (b *Baton) gatherUncommittedTxs(ctx context.Context) []*proto.SignedTransaction {
	// pool all available transactions (our own + from batches) excluding already committed ones
	uncommittedTxs := []*proto.SignedTransaction{}
	for _, stx := range b.handler.ListTxs(ctx) {
		hash := sha256.Sum256(xdrutil.MarshalXdr(stx))
		if !b.handler.IsCommittedTx(ctx, hash) {
			uncommittedTxs = append(uncommittedTxs, stx)
		}
	}
	return uncommittedTxs
}

func (b *Baton) nominate(ctx context.Context, tip sdk.Tip, uncommittedTxs []*proto.SignedTransaction) (*proto.Proposal, proto.BlockID, error) {

	must.Assertf(ctx, len(uncommittedTxs) > 0, "nominating no uncommited txs")

	// prioritize and pick the top, applying any limits on transactions per block
	txsToNominate, rejectedTxs := b.handler.PrioritizeRejectTxs(ctx, uncommittedTxs)
	for _, rtx := range rejectedTxs {
		hash := sha256.Sum256(xdrutil.MarshalXdr(&rtx.Transaction))
		b.handler.AddCommittedTx(ctx, hash) // adding to committed so they are not tried again
		b.handler.RemoveTx(ctx, hash)
		b.handler.RejectedTx(ctx, rtx)
	}
	if len(txsToNominate) == 0 {
		return nil, 0, ErrNothingToNominate
	}

	// start working on proposing a block, while being ready to cancel if a block is externalized by someone else
	nom, err := nominate.PrepareNomination(ctx, b.handler, tip, txsToNominate)
	if err != nil {
		// XXX: ctx may be cancelled
		// XXX: ErrNoExecs
		return nil, 0, err
	}

	// construct proposed block
	batch, batchHash := protoutil.PrepareBatch(nom.ProposedTxs)
	proposal := &proto.Proposal{
		Block_height:                 nom.BlockHeight,
		Block_hash:                   nom.BlockHash,
		Prev_block_hash:              nom.PrevBlockHash,
		Proposed_transaction_batches: []proto.BatchHash{batchHash},
	}

	b.handler.Emit(ctx, sdk.EventNominating{Tip: tip, NumTx: len(txsToNominate), Proposal: proposal})

	// save batches
	b.handler.AddBatch(ctx, batch)

	// disseminate batches; this is non-blocking (no network ops)
	b.handler.Disseminate(ctx, &proto.BatchMessage{Batch_transactions: batch})

	// nominate to consensus
	prevSummary := proto.PrevProposalSummary{
		Block_hash: nom.PrevBlockHash,
	}
	// block until nomination is over
	b.handler.Nominate(ctx, nom.BlockHeight, proposal, &prevSummary) // ignore result
	b.handler.Emit(ctx, sdk.EventNominated{Tip: tip, NumTx: len(txsToNominate), Proposal: proposal})

	return proposal, nom.BlockID, nil
}
