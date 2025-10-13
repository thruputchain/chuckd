package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/baton/batonapi"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) validate(
	ctx context.Context,
	onto proto.DbTip,
	nomination *proto.Nomination,
) (proto.DbTip, error) {

	for {
		schedValidResult, err := b.handler.Schedulercoord_validate(ctx, onto, *nomination)
		if err != nil {
			liblog.Infof("cannot validate, scheduler error (%v)", err)
			panic("scheduler failed on validation")
		}
		switch schedValidResult.Outcome {

		case proto.SchedulerValidationOK:
			return *schedValidResult.Db_tip(), nil

		case proto.SchedulerValidationInvalid:
			return proto.DbTip{}, batonapi.ErrNotValid

		case proto.SchedulerValidationMissingTxs:
			missing := proto.MissingTransactions{
				Transaction_hashes: *schedValidResult.Missing_tx_hashes(),
				Block_hash:         nomination.Block_hash,
			}
			err := b.repairMissingTxs(ctx, nomination.Proposer, missing)
			if err != nil { // err can only be context cancelled
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					//XXX: check that upstream caller handles this error correctly
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

		}
	}
}

func (b *Baton) repairMissingTxs(
	ctx context.Context,
	proposer proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) error {

	stxs, err := b.handler.RepairMissingTxs(ctx, proposer, missing)
	if err != nil {
		return err // this can only ever be context cancelled
	}

	err = b.handler.Schedulercoord_submit_unvetted(ctx, stxs)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		must.NoError(ctx, err)
	}

	return nil
}
