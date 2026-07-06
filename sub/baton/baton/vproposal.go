package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) validateProposal(
	ctx context.Context,
	onto proto.DbTip,
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
) (proto.DbTip, error) {
	header := &value.Proposal_header
	proposal := proto.Proposal{
		Header:       *header,
		Transactions: basis.Proposal_txs,
	}
	numTxs := xdrutil.TokenSizeOfBatch(basis.Proposal_txs.Hash)

	eventValidating := batonkit.EventValidating{
		Onto:           onto,
		NumTx:          numTxs,
		Height:         header.Block_height,
		ValueHash:      protoutil.HashValueToProto(value),
		ConsensusValue: value,
		ConsensusBasis: basis,
	}
	liblog.Infof(ctx, "%v", eventValidating)
	b.handler.Emit(ctx, eventValidating)

	defer func() {
		eventValidated := batonkit.EventValidated{
			Onto:           onto,
			NumTx:          numTxs,
			Height:         header.Block_height,
			ValueHash:      protoutil.HashValueToProto(value),
			ConsensusValue: value,
			ConsensusBasis: basis,
		}
		liblog.Infof(ctx, "%v", eventValidated)
		b.handler.Emit(ctx, eventValidated)
	}()

	for {
		liblog.Infof(ctx, "scheduler validating: height=%d; value=%s",
			header.Block_height, protoutil.ValueDisplay(value))
		schedValidResult, err := b.handler.Schedulercoord_validate_block(ctx, onto.Id, proposal)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return proto.DbTip{}, err
			}
			liblog.Infof(ctx, "cannot validate, scheduler error (%v)", err)
			panic("scheduler failed on validation")
		}
		switch schedValidResult.Outcome {

		case proto.SchedulerValidationOK:
			liblog.Debugf(ctx, "scheduler validated: height=%d; value=%s",
				header.Block_height, protoutil.ValueDisplay(value))

			vblock := schedValidResult.Block()

			if !protoutil.HeaderEquals(&vblock.Db_tip.Value.Proposal_header, header) {
				liblog.Fatalf(ctx, "proposal header mismatch: scheduler computed %+v, proposal claims %+v",
					vblock.Db_tip.Value.Proposal_header, *header)
				panic("scheduler bug, proposal header mismatch")
			}

			cb := &proto.CatchupBlock{
				Consensus_basis:     *basis,
				Signed_transactions: vblock.Signed_transactions,
			}
			b.handler.CBPoolAdd(ctx, *cb, *value)

			return vblock.Db_tip, nil

		case proto.SchedulerValidationInvalid:
			liblog.Infof(ctx, "scheduler invalidated: height=%d; value=%s",
				header.Block_height, protoutil.ValueDisplay(value))
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationTooLateToValidate:
			liblog.Infof(ctx, "scheduler is too late to validate: height=%d; value=%s",
				header.Block_height, protoutil.ValueDisplay(value))
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationUnknownParentBlock:
			liblog.Infof(ctx, "scheduler cannot validate due to missing parent block: height=%d; value=%s",
				header.Block_height, protoutil.ValueDisplay(value))
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationMissingTxs:
			liblog.Infof(ctx, "scheduler is missing transactions: height=%d; value=%s",
				header.Block_height, protoutil.ValueDisplay(value))
			missing := proto.MissingTransactions{
				Consensus_value:    *value,
				Transaction_hashes: *schedValidResult.Missing_tx_hashes(),
			}
			err := b.repairMissingTxs(ctx, basis.Value.Proposer, missing)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return proto.DbTip{}, err
				}
				must.NoError(ctx, err)
			}

		case proto.SchedulerValidationDbFailed:
			liblog.Fatalf(ctx, "cannot validate, db failed")

		case proto.SchedulerValidationExecutorFailed:
			liblog.Fatalf(ctx, "cannot validate, executor failed")

		default:
			liblog.Fatalf(ctx, "unknown scheduler validation outcome: %v", schedValidResult.Outcome)
		}
	}
}

func (b *Baton) repairMissingTxs(
	ctx context.Context,
	proposer proto.ValidatorID,
	missing proto.MissingTransactions,
) (err error) {
	height := missing.Consensus_value.Proposal_header.Block_height
	liblog.Debugf(ctx, "repairing %d missing transactions: height=%d; value=%s",
		len(missing.Transaction_hashes), height, protoutil.ValueDisplay(&missing.Consensus_value))
	defer func() {
		liblog.Debugf(ctx, "repaired %d missing transactions: height=%d; value=%s; error=%v",
			len(missing.Transaction_hashes), height, protoutil.ValueDisplay(&missing.Consensus_value), err)
	}()

	stxs, err := b.handler.RepairMissingTxs(ctx, proposer, missing)
	if err != nil {
		must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		return err
	}

	batch := xdrutil.ParallelizeSignedTransactionBatch(b.batchRepairM, systemfixed.RepairTxBatchMaxSegments, stxs)
	err = b.handler.Schedulercoord_submit_vetted(ctx, proposer, batch)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		must.NoError(ctx, err)
	}
	return nil
}
