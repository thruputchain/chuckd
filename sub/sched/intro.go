package sched

import (
	"context"
	"fmt"

	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libnet"
	proto "github.com/groundhogchain/protocol-go"
)

func ConnectToScheduler(
	ctx context.Context,
	net libnet.Dialer,
	schedAddr string,
	coordInAddr string,
) (SchedulerCoordConn, error) {

	schedConn, err := ConnectPoolToScheduler(ctx, net, schedAddr)
	if err != nil {
		return nil, err
	}

	err = intro(ctx, schedConn, coordInAddr)
	if err != nil {
		liblog.Errorf(ctx, "intro to scheduler failed (%v)", err)
		schedConn.Close()
		return nil, err
	}

	liblog.Infof(ctx, "connected to scheduler at %v", schedAddr)

	return schedConn, nil
}

func intro(
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
