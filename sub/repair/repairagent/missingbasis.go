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

// RepairMissingBasis ensures that the returned basis matches the desired hash.
func (x *RepairAgent) RepairMissingBasis(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing proto.ConsensusValueCue,
) (*proto.ConsensusBasis, error) {

	backoff := netutil.NewBackoff(x.backoffCfg, librand.NewRand())

attempting:
	for {
		basis, err := x.attemptRepairMissingBasis(ctx, from, &missing)
		if err != nil {
			liblog.Errorf(ctx, "failed attempt to repair missing basis at height %d from %s: %v",
				missing.Block_height, protoutil.VPublicShortString(from), err)
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
		x.cvpool.CVPoolAdd(ctx, *basis, missing) // repaired cvs should go in the cvpool to be accessible later
		return basis, nil
	}
}

func (x *RepairAgent) attemptRepairMissingBasis(
	ctx context.Context,
	from proto.ValidatorPublicInfo,
	missing *proto.ConsensusValueCue,
) (*proto.ConsensusBasis, error) {

	c, err := repairkit.DialRepairClient(ctx, x.net, from.Address)
	if err != nil {
		return nil, fmt.Errorf("dialing repair endpoint: %w", err)
	}
	defer c.Close()

	reply, err := c.Repair_missing_basis(ctx, *missing)
	if err != nil {
		return nil, fmt.Errorf("calling Repair_missing_basis: %w", err)
	}

	if reply == nil {
		return nil, fmt.Errorf("no repaired consensus value returned, cvh %s, vid %s",
			libxdr.HashString(missing.Cv_hash), protoutil.VIDString(protoutil.PublicVID(from)))
	}

	// verify returned basis matches hash
	if !protoutil.IsSaneConsensusBasis(reply) {
		return nil, fmt.Errorf("repaired consensus value is not sane")
	}
	if !protoutil.CVCEquals(missing, protoutil.CVC(reply)) {
		return nil, fmt.Errorf("repaired consensus value cue mismatch")
	}

	return reply, nil
}
