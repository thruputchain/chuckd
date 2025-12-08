package repairagent

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/libtext"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/librand"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/sub/repair/repairkit"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

// RepairMissingCV ensures that the returned cv match the desired hash.
func (x *RepairAgent) RepairMissingCV(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.ConsensusValueHash,
	height proto.BlockHeight,
) (*proto.ConsensusValue, error) {

	backoff := netutil.NewBackoff(x.backoffCfg, librand.NewRand())

attempting:
	for {
		cv, err := x.attemptRepairMissingCV(ctx, from, missing, height)
		if err != nil {
			delay, canRetry := backoff.NextDelay()
			must.Assert(ctx, canRetry)
			tmo := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-tmo.C:
				continue attempting
			}
		}
		x.cvpool.ConsensusValuePoolAdd(ctx, *cv) // repaired cvs should go in the cvpool to be accessible later
		return cv, nil
	}
}

func (x *RepairAgent) attemptRepairMissingCV(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.ConsensusValueHash,
	height proto.BlockHeight,
) (*proto.ConsensusValue, error) {

	c, err := repairkit.DialRepairClient(ctx, x.net, from.Address)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	reply, err := c.Repair_missing_cv(ctx, missing, height)
	if err != nil {
		return nil, err
	}

	if reply == nil {
		liblog.Errorf("no repaired consensus value returned, cvh %s, vid %s",
			libtext.Display(missing), protoutil.VIDString(protoutil.PublicVID(from)))
		return nil, fmt.Errorf("no repaired consensus value returned")
	}

	// verify returned cv matches hash
	if missing != protoutil.HashCV(x.cvM, reply) {
		liblog.Errorf("repaired consensus value hash mismatch")
		return nil, fmt.Errorf("repaired consensus value hash mismatch")
	}

	return reply, nil
}
