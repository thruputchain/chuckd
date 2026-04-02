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
	cvh proto.ConsensusValueHash,
) (proto.DbTip, error) {

	nomination := cv.Nomination

	numTxs := xdrutil.SizeOfBatch(nomination.Transactions_hash)

	eventValidating := batonkit.EventValidating{
		Onto:              onto,
		NumTx:             numTxs,
		Height:            nomination.Block_height,
		BlockHash:         nomination.Block_hash,
		CVHash:            cvh,
		DbgConsensusValue: cv,
	}
	liblog.Infof("%v", eventValidating)
	b.handler.Emit(ctx, eventValidating)

	defer func() {
		eventValidated := batonkit.EventValidated{
			Onto:              onto,
			NumTx:             numTxs,
			Height:            nomination.Block_height,
			BlockHash:         nomination.Block_hash,
			CVHash:            cvh,
			DbgConsensusValue: cv,
		}
		liblog.Infof("%v", eventValidated)
		b.handler.Emit(ctx, eventValidated)
	}()

	for {
		liblog.Infof("scheduler validating: height=%d; cv_hash=%s",
			nomination.Block_height, libxdr.HashString(cvh))
		schedValidResult, err := b.handler.Schedulercoord_validate(ctx, onto, nomination)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return proto.DbTip{}, err
			}
			liblog.Infof("cannot validate, scheduler error (%v)", err)
			panic("scheduler failed on validation")
		}
		switch schedValidResult.Outcome {

		case proto.SchedulerValidationOK:
			liblog.Debugf("scheduler validated: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvh))

			vblock := schedValidResult.Block()

			cb := &proto.CatchupBlock{
				Consensus_value:     *cv,
				Signed_transactions: vblock.Signed_transactions,
			}

			// kick-off an optimistic pre-persist of the catchup block;
			// this will be committed/persisted by commit if the block is committed.
			b.handler.CatchupBlockPoolAdd(ctx, *cb, cvh)

			return vblock.Db_tip, nil

		case proto.SchedulerValidationInvalid:
			liblog.Infof("scheduler invalidated: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvh))
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationTooLateToValidate:
			liblog.Infof("scheduler is too late to validate: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvh))
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationMissingTxs:
			liblog.Infof("scheduler is missing transactions: height=%d; cv_hash=%s",
				nomination.Block_height, libxdr.HashString(cvh))
			missing := proto.MissingTransactions{
				Transaction_hashes: *schedValidResult.Missing_tx_hashes(),
				Cv_hash:            cvh,
				Block_hash:         nomination.Block_hash,
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
			liblog.Fatalf("cannot validate, db failed")

		case proto.SchedulerValidationExecutorFailed:
			// see https://github.com/groundhogchain/hogd/issues/541
			liblog.Fatalf("cannot validate, executor failed")

		default:
			liblog.Fatalf("unknown scheduler validation outcome: %v", schedValidResult.Outcome)

		}
	}
}

func (b *Baton) repairMissingTxs(
	ctx context.Context,
	proposer proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) error {

	liblog.Debugf("repairing %d missing transactions", len(missing.Transaction_hashes))

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
