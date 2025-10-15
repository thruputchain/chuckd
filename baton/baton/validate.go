package baton

import (
	"context"
	"errors"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) getValidated(hash proto.BlockHash) *proto.BlockID {
	v, ok := b.validated[hash]
	if !ok {
		return nil
	}
	return &v.BlockID
}

func (b *Baton) validate(
	ctx context.Context,
	onto proto.DbTip,
	nomination *proto.Nomination,
) (proto.DbTip, error) {

	b.handler.Emit(ctx, batonkit.EventValidating{
		Onto:   onto,
		NumTx:  len(nomination.Transaction_hashes),
		Height: nomination.Block_height,
		Hash:   nomination.Block_hash,
	})

	defer func() {
		b.handler.Emit(ctx, batonkit.EventValidated{
			Onto:   onto,
			NumTx:  len(nomination.Transaction_hashes),
			Height: nomination.Block_height,
			Hash:   nomination.Block_hash,
		})
	}()

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
			return proto.DbTip{}, batonkit.ErrNotValid

		case proto.SchedulerValidationMissingTxs:
			missing := proto.MissingTransactions{
				Transaction_hashes: *schedValidResult.Missing_tx_hashes(),
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

		}
	}
}

func (b *Baton) repairMissingTxs(
	ctx context.Context,
	proposer proto.ValidatorPublicInfo,
	missing proto.MissingTransactions,
) error {

	liblog.Debugf("baton repairing %d missing transactions", len(missing.Transaction_hashes))

	time.Sleep(time.Second / 2)
	return nil

	//XXX: temporarily disable the code below

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
