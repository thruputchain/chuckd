package repairagent

import (
	"context"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/sub/repair/repairkit"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/librand"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
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
			liblog.Errorf(ctx, "failed attempt to repair missing cv at height %d from %s: %v", height, protoutil.VPublicShortString(from), err)
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
		x.cvpool.CVPoolAdd(ctx, *cv, missing) // repaired cvs should go in the cvpool to be accessible later
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
		return nil, fmt.Errorf("dialing repair endpoint: %w", err)
	}
	defer c.Close()

	reply, err := c.Repair_missing_cv(ctx, missing, height)
	if err != nil {
		return nil, fmt.Errorf("calling Repair_missing_cv: %w", err)
	}

	if reply == nil {
		return nil, fmt.Errorf("no repaired consensus value returned, cvh %s, vid %s",
			libxdr.HashString(missing), protoutil.VIDString(protoutil.PublicVID(from)))
	}

	// verify returned cv matches hash
	if !protoutil.IsSaneConsensusValue(reply) {
		return nil, fmt.Errorf("repaired consensus value is not sane")
	}
	if missing != protoutil.HashCVFast(reply) {
		return nil, fmt.Errorf("repaired consensus value hash mismatch")
	}

	return reply, nil
}
