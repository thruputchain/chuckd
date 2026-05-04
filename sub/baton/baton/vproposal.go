package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	"github.com/groundhogchain/chuckd/system/systemfixed"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

// XXX: validateProposal upgrades; update validation cache with regards to upgrades (now only caching block proposal validity)
func (b *Baton) validateProposal(
	ctx context.Context,
	onto proto.DbTip,
	cv *proto.ConsensusValue,
	cvc proto.ConsensusValueCue,
) (proto.DbTip, error) {

	nomination := cv.Nomination

	numTxs := xdrutil.TokenSizeOfBatch(nomination.Transactions_hash)

	eventValidating := batonkit.EventValidating{
		Onto:              onto,
		NumTx:             numTxs,
		Height:            nomination.Block_height,
		BlockHash:         nomination.Block_hash,
		CVHash:            cvc.Cv_hash,
		DbgConsensusValue: cv,
	}
	liblog.Infof(ctx, "%v", eventValidating)
	b.handler.Emit(ctx, eventValidating)

	defer func() {
		eventValidated := batonkit.EventValidated{
			Onto:              onto,
			NumTx:             numTxs,
			Height:            nomination.Block_height,
			BlockHash:         nomination.Block_hash,
			CVHash:            cvc.Cv_hash,
			DbgConsensusValue: cv,
		}
		liblog.Infof(ctx, "%v", eventValidated)
		b.handler.Emit(ctx, eventValidated)
	}()

	for {
		liblog.Infof(ctx, "scheduler validating: height=%d; cv_hash=%s",
			nomination.Block_height, libxdr.HashString(cvc.Cv_hash))
		schedValidResult, err := b.handler.Schedulercoord_validate(ctx, onto, nomination)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return proto.DbTip{}, err
			}
			liblog.Infof(ctx, "cannot validate, scheduler error (%v)", err)
			panic("scheduler failed on validation")
		}
		switch schedValidResult.Outcome {

		case proto.SchedulerValidationOK:
			liblog.Debugf(ctx, "scheduler validated: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvc.Cv_hash))

			vblock := schedValidResult.Block()

			if vblock.Db_tip.Block_hash != nomination.Block_hash {
				liblog.Fatalf(ctx, "block hash mismatch: scheduler computed %s, nomination claims %s",
					libxdr.HashString(vblock.Db_tip.Block_hash),
					libxdr.HashString(nomination.Block_hash))
				panic("scheduler bug, block hash mismatch")
			}

			cb := &proto.CatchupBlock{
				Consensus_value:     *cv,
				Signed_transactions: vblock.Signed_transactions,
			}

			// kick-off an optimistic pre-persist of the catchup block;
			// this will be committed/persisted by commit if the block is committed.
			b.handler.CBPoolAdd(ctx, *cb, cvc)

			return vblock.Db_tip, nil

		case proto.SchedulerValidationInvalid:
			liblog.Infof(ctx, "scheduler invalidated: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvc.Cv_hash))
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationTooLateToValidate:
			liblog.Infof(ctx, "scheduler is too late to validate: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvc.Cv_hash))
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationUnknownParentBlock:
			liblog.Infof(ctx, "scheduler cannot validate due to missing parent block: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvc.Cv_hash))
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationMissingTxs:
			liblog.Infof(ctx, "scheduler is missing transactions: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvc.Cv_hash))
			missing := proto.MissingTransactions{
				Cue:                cvc,
				Transaction_hashes: *schedValidResult.Missing_tx_hashes(),
			}
			err := b.repairMissingTxs(ctx, nomination.Proposer, missing)
			if err != nil { // err can only be context cancelled
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return proto.DbTip{}, err
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
	proposer proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) (err error) {

	liblog.Debugf(ctx, "repairing %d missing transactions: height=%d; cv_hash=%s; block_hash=%s",
		len(missing.Transaction_hashes), missing.Cue.Block_height, libxdr.HashString(missing.Cue.Cv_hash),
		libxdr.HashString(missing.Cue.Block_hash))
	defer func() {
		liblog.Debugf(ctx, "repaired %d missing transactions: height=%d; cv_hash=%s; block_hash=%s; error=%v",
			len(missing.Transaction_hashes), missing.Cue.Block_height,
			libxdr.HashString(missing.Cue.Cv_hash), libxdr.HashString(missing.Cue.Block_hash), err)

	}()

	stxs, err := b.handler.RepairMissingTxs(ctx, proposer, missing)
	if err != nil {
		// this can only ever be context cancelled
		must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		return err
	}

	batch := xdrutil.ParallelizeSignedTransactionBatch(b.batchRepairM, systemfixed.RepairTxBatchMaxSegments, stxs)
	err = b.handler.Schedulercoord_submit_vetted(ctx, protoutil.PublicVID(proposer), batch)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		must.NoError(ctx, err)
	}

	return nil
}
