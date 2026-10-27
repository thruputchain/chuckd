package baton

import (
	"context"
	"errors"

	"github.com/thruputchain/chuckd/sub/baton/batonkit"
	"github.com/thruputchain/chuckd/system/systemfixed"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/chuckd/util/xdrutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// XXX: validateProposal upgrades; update validation cache with regards to upgrades (now only caching block proposal validity)
func (b *Baton) validateProposal(
	ctx context.Context,
	onto proto.DbTip,
	basis *proto.ConsensusBasis,
	value *proto.ConsensusValue,
) (proto.BlockID, error) {

	proposal := protoutil.ProposalFromBasis(basis)
	height := proposal.Header.Block_height

	numTxs := xdrutil.TokenSizeOfBatch(proposal.Transactions.Signed_hash)

	eventValidating := batonkit.EventValidating{
		Onto:           onto,
		NumTx:          numTxs,
		Height:         height,
		ConsensusValue: value,
		ConsensusBasis: basis,
	}
	liblog.Debugf(ctx, "%v", eventValidating)
	b.handler.Emit(ctx, eventValidating)

	defer func() {
		eventValidated := batonkit.EventValidated{
			Onto:           onto,
			NumTx:          numTxs,
			Height:         height,
			ConsensusValue: value,
			ConsensusBasis: basis,
		}
		liblog.Debugf(ctx, "%v", eventValidated)
		b.handler.Emit(ctx, eventValidated)
	}()

	for {
		liblog.Debugf(ctx, "scheduler validating: height=%d; value=%s",
			height, protoutil.ValueDisplay(value))
		schedValidResult, err := b.handler.Schedulercoord_validate_block(ctx, onto.Id, *proposal)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return 0, err
			}
			liblog.Infof(ctx, "cannot validate, scheduler error (%v)", err)
			panic("scheduler failed on validation")
		}
		switch schedValidResult.Outcome {

		case proto.SchedulerValidationOK:
			liblog.Debugf(ctx, "scheduler validated: height=%d; value=%s",
				height, protoutil.ValueDisplay(value))

			vblock := schedValidResult.Block()

			cb := &proto.CatchupBlock{
				Consensus_basis:     *basis,
				Signed_transactions: vblock.Signed_transactions,
			}

			// kick-off an optimistic pre-persist of the catchup block;
			// this will be committed/persisted by commit if the block is committed.
			b.handler.CBPoolAdd(ctx, *cb, *value)

			return vblock.Block_id, nil

		case proto.SchedulerValidationInvalid:
			liblog.Infof(ctx, "scheduler invalidated: height=%d; value=%s",
				height, protoutil.ValueDisplay(value))
			return 0, batonkit.ErrNotValid

		case proto.SchedulerValidationTooLateToValidate:
			liblog.Infof(ctx, "scheduler is too late to validate: height=%d; value=%s",
				height, protoutil.ValueDisplay(value))
			return 0, batonkit.ErrNotValid

		case proto.SchedulerValidationUnknownParentBlock:
			liblog.Infof(ctx, "scheduler cannot validate due to missing parent block: height=%d; value=%s",
				height, protoutil.ValueDisplay(value))
			return 0, batonkit.ErrNotValid

		case proto.SchedulerValidationMissingTxs:
			liblog.Infof(ctx, "scheduler is missing transactions: height=%d; value=%s",
				height, protoutil.ValueDisplay(value))
			missing := proto.MissingTransactions{
				Consensus_value:  *value,
				Signed_tx_hashes: *schedValidResult.Missing_signed_tx_hashes(),
			}
			err := b.repairMissingTxs(ctx, protoutil.ValueProposer(value), missing)
			if err != nil { // err can only be context cancelled
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return 0, err
				}
				must.NoError(ctx, err)
			}

		case proto.SchedulerValidationDbFailed:
			// see https://github.com/groundhogchain/hogd/issues/541
			liblog.Fatalf(ctx, "cannot validate, db failed")

		case proto.SchedulerValidationExecutorFailed:
			// see https://github.com/groundhogchain/hogd/issues/541
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

	liblog.Debugf(ctx, "repairing %d missing transactions: height=%d; value=%s",
		len(missing.Signed_tx_hashes), protoutil.ValueHeight(&missing.Consensus_value),
		protoutil.ValueDisplay(&missing.Consensus_value))
	defer func() {
		liblog.Debugf(ctx, "repaired %d missing transactions: height=%d; value=%s; error=%v",
			len(missing.Signed_tx_hashes), protoutil.ValueHeight(&missing.Consensus_value),
			protoutil.ValueDisplay(&missing.Consensus_value), err)

	}()

	stxs, err := b.handler.RepairMissingTxs(ctx, proposer, missing)
	if err != nil {
		// this can only ever be context cancelled
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
