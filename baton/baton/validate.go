package baton

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/liblog"
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
