package repairagent

import (
	"context"
	"fmt"
	"time"

	"github.com/thruputchain/chuckd/sub/repair/repairkit"
	"github.com/thruputchain/chuckd/util/netutil"
	"github.com/thruputchain/chuckd/util/protoutil"
	"github.com/thruputchain/lib-go/liblog"
	"github.com/thruputchain/lib-go/librand"
	"github.com/thruputchain/lib-go/must"
	proto "github.com/thruputchain/thruput-protocol-go"
)

// RepairMissingBasis ensures that the returned basis matches the desired hash.
func (x *RepairAgent) RepairMissingBasis(
	ctx context.Context,
	from proto.ValidatorID,
	missing proto.ConsensusValue,
) (*proto.ConsensusBasis, error) {

	backoff := netutil.NewBackoff(x.backoffCfg, librand.NewRand())

attempting:
	for {
		basis, err := x.attemptRepairMissingBasis(ctx, from, &missing)
		if err != nil {
			liblog.Errorf(ctx, "failed attempt to repair missing basis at height %d from %s: %v",
				protoutil.ValueHeight(&missing), protoutil.VIDString(from), err)
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
	from proto.ValidatorID,
	missing *proto.ConsensusValue,
) (*proto.ConsensusBasis, error) {

	address, err := x.router.ResolveAddress(from)
	if err != nil {
		return nil, fmt.Errorf("resolving address for %s: %w", protoutil.VIDString(from), err)
	}

	c, err := repairkit.DialRepairClient(ctx, x.net, address)
	if err != nil {
		return nil, fmt.Errorf("dialing repair endpoint: %w", err)
	}
	defer c.Close()

	reply, err := c.Repair_missing_proposal(ctx, *missing)
	if err != nil {
		return nil, fmt.Errorf("calling Repair_missing_proposal: %w", err)
	}

	if reply == nil {
		return nil, fmt.Errorf("no repaired proposal returned, value %s, vid %s",
			protoutil.ValueDisplay(missing), protoutil.VIDString(from))
	}

	// verify the returned proposal header agrees with the requested value
	if !protoutil.HeaderEquals(&reply.Header, &missing.Proposal_header) {
		return nil, fmt.Errorf("repaired proposal header/value mismatch")
	}

	// reconstruct the basis from the requested value and the repaired proposal txs,
	// and verify the proposal txs fast-hash to the proposal_txs_hash in the value
	basis := &proto.ConsensusBasis{
		Value:        *missing,
		Proposal_txs: reply.Transactions,
	}
	if !protoutil.IsValidBasis(basis) {
		return nil, fmt.Errorf("repaired consensus basis is not valid")
	}

	return basis, nil
}
