package baton

import (
	"context"
	"errors"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (x scpHandler) ValidateCueValue(ctx context.Context, slot scpkit.Slot, value scpkit.Value, nomination bool) (level scpkit.ValidationLevel) {
	<-x.baton.ready

	kind := "ballot"
	if nomination {
		kind = "nomination"
	}
	liblog.Infof("baton validate %s value for slot %d received", kind, slot)
	defer func() {
		liblog.Infof("baton validate %s value for slot %d completed with %v", kind, slot, scpkit.ValidationLevelString(level))
	}()

	var cvc proto.ConsensusValueCue
	err := xdrutil.UnmarshalXdr(&cvc, value)
	if err != nil {
		return scpkit.InvalidValue
	}

	reply := make(chan scpkit.ValidationLevel, 1)
	go x.baton.validateCueValue(ctx, &cvc, reply)
	level = <-reply
	return
}

func (b *Baton) validateCueValue(
	ctx context.Context,
	cvc *proto.ConsensusValueCue,
	reply chan<- scpkit.ValidationLevel,
) {

	// XXX: how long should be wait before cancelling fetchCV
	cv, err := b.fetchCV(ctx, cvc)
	if err != nil {
		liblog.Infof("cannot validate, fetch consensus value (%v)", err)
		reply <- scpkit.InvalidValue
		return
	}

	b.validateConsensusValue(ctx, cv, reply)
}

func (b *Baton) fetchCV(ctx context.Context, cvc *proto.ConsensusValueCue) (*proto.ConsensusValue, error) {

	// lookup in cvpool
	cv := b.handler.LookupConsensusValue(ctx, cvc.Cv_hash)
	if cv != nil {
		return cv, nil
	}

	cvReturn := make(chan *proto.ConsensusValue, 2)

	// try repairing the missing cv
	// this goroutine keeps on trying unless cancelled
	go func() {
		cv, err := b.handler.RepairMissingCV(ctx, cvc.Repair_fallback, cvc.Cv_hash)
		if err != nil {
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		cvReturn <- cv
	}()

	// try getting the cv from the pool, if it arrives before the repair
	// this goroutine keeps on trying unless cancelled
	go func() {
		cv, err := b.handler.WaitForConsensusValue(ctx, cvc.Cv_hash)
		if err != nil {
			must.Assert(ctx, errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
			return
		}
		cvReturn <- cv
	}()

	cv, ok := <-cvReturn
	if ok {
		return cv, nil
	}
	return nil, context.Canceled
}
