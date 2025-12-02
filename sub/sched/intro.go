package sched

import (
	"context"
	"fmt"

	proto "github.com/groundhogchain/protocol-go"
)

func Intro(
	ctx context.Context,
	conn SchedulerCoordConn,
	address_for_sched string,
) error {
	outcome, err := conn.Schedulercoord_intro_coord(ctx, proto.CoordCard{
		Protocol:          proto.HogProtocol(),
		Address_for_sched: address_for_sched,
	})
	if err != nil {
		return err
	}
	switch outcome {
	case proto.CoordinatorRegistrationOK:
		return nil
	case proto.CoordinatorRegistrationInvalidProtocol:
		return fmt.Errorf("coordinator protocol not accepted by scheduler")
	}
	panic("unreachable")
}
