package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (b *Baton) hashCV(cv *proto.ConsensusValue) proto.ConsensusValueHash {
	return protoutil.HashCV(b.cvM, cv)
}

func (b *Baton) isAlreadyValidated(cv *proto.ConsensusValue) bool {
	_, alreadyValidated := b.validated[b.hashCV(cv)]
	return alreadyValidated
}

func (b *Baton) setValidated(cv *proto.ConsensusValue, blockID proto.BlockID) {
	b.validated[b.hashCV(cv)] = validated{
		ConsensusValue: cv,
		BlockID:        blockID,
	}
}

func (b *Baton) getValidated(cv *proto.ConsensusValue) *proto.BlockID {
	v, ok := b.validated[b.hashCV(cv)]
	if !ok {
		return nil
	}
	return &v.BlockID
}

// XXX: validate timestamp
// XXX: validate upgrades; update validation cache with regards to upgrades (now only caching block proposal validity)
func (b *Baton) validate(
	ctx context.Context,
	onto proto.DbTip,
	cv *proto.ConsensusValue,
) (proto.DbTip, error) {

	b.handler.Emit(ctx, batonkit.EventValidating{
		Onto:   onto,
		NumTx:  len(cv.Nomination.Transaction_hashes),
		Height: cv.Nomination.Block_height,
		Hash:   cv.Nomination.Block_hash,
	})

	defer func() {
		b.handler.Emit(ctx, batonkit.EventValidated{
			Onto:   onto,
			NumTx:  len(cv.Nomination.Transaction_hashes),
			Height: cv.Nomination.Block_height,
			Hash:   cv.Nomination.Block_hash,
		})
	}()

	for {
		schedValidResult, err := b.handler.Schedulercoord_validate(ctx, onto, cv.Nomination)
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
				Block_hash:         cv.Nomination.Block_hash,
			}
			err := b.repairMissingTxs(ctx, cv.Nomination.Proposer, missing)
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

	stxs, err := b.handler.RepairMissingTxs(ctx, proposer, missing)
	if err != nil {
		// this can only ever be context cancelled
		must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
		return err
	}

	err = b.handler.Schedulercoord_submit_vetted(ctx, protoutil.PublicVID(proposer), stxs)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		must.NoError(ctx, err)
	}

	return nil
}
