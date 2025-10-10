package baton

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) validate(
	ctx context.Context,
	onto proto.DbTip,
	nomination *proto.Nomination,
) (proto.DbTip, error) {

	schedValidResult, err := b.handler.Schedulercoord_validate(ctx, onto, *nomination)
	if err != nil {
		liblog.Infof("cannot validate, scheduler error (%v)", err)
		panic("scheduler failed on validation")
	}
	switch schedValidResult.Outcome {

	case proto.SchedulerValidationOK:
		return *schedValidResult.Db_tip(), nil

	case proto.SchedulerValidationInvalid:
		return proto.DbTip{}, fmt.Errorf("block proposal not valid")

	case proto.SchedulerValidationMissingTxs:
		panic("XXX")

	case proto.SchedulerValidationDbFailed:
		// see https://github.com/groundhogchain/hogd/issues/541
		liblog.Fatalf("cannot validate, db failed")

	case proto.SchedulerValidationExecutorFailed:
		// see https://github.com/groundhogchain/hogd/issues/541
		liblog.Fatalf("cannot validate, executor failed")

	}
	panic("bug")
}

func (b *Baton) vbNomination(vb *batonapi.ValidatableBlock) *proto.Nomination {

	hashes := make([]proto.SignedTransactionHash, len(vb.ProposedTransactions))
	txs := make([]proto.NominatedTransaction, len(vb.ProposedTransactions))
	for i, ptx := range vb.ProposedTransactions {
		hashes[i] = protoutil.HashSignedTx(b.txM, &ptx.Signed_tx)
		txs[i] = proto.NominatedTransaction{
			Nondeterministic_proposal_data: ptx.Nondeterministic_proposal_data,
		}
	}

	return &proto.Nomination{
		Proposer:                b.handler.Signer(),
		Block_height:            vb.BlockHeight,
		Block_hash:              vb.BlockHash,
		Prev_block_hash:         vb.PrevBlockHash,
		Transaction_hashes:      hashes,
		Transaction_nominations: txs,
	}
}
