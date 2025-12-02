package sched

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	proto "github.com/groundhogchain/protocol-go"
)

func ConnectToScheduler(
	ctx context.Context,
	net libnet.AddressDialer,
	schedAddr string,
	coordInAddr string,
) (SchedulerCoordConn, error) {

	var err error
	var schedConn SchedulerCoordConn

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		schedConn, err = DialSchedulerCoordClient(ctx, net, schedAddr)
		if err != nil {
			liblog.Infof("connecting to scheduler failed (%v), retrying in 1 sec", err)
			time.Sleep(time.Second)
			continue
		}

		break
	}

	err = intro(ctx, schedConn, coordInAddr)
	if err != nil {
		liblog.Errorf("intro to scheduler failed (%v)", err)
		schedConn.Close()
		return nil, err
	}

	liblog.Infof("connected to scheduler at %v", schedAddr)

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
