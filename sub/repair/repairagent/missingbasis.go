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

// RepairMissingBasis repairs the proposal committed to by the missing value
// and assembles the corresponding consensus basis.
func (x *RepairAgent) RepairMissingBasis(
	ctx context.Context,
	from proto.ValidatorID,
	missing proto.ConsensusValue,
) (*proto.ConsensusBasis, error) {

	backoff := netutil.NewBackoff(x.backoffCfg, librand.NewRand())

attempting:
	for {
		basis, err := x.attemptRepairMissingProposal(ctx, from, &missing)
		if err != nil {
			liblog.Errorf(ctx, "failed attempt to repair missing proposal at height %d from %s: %v",
				missing.Proposal_header.Block_height, protoutil.VIDString(from), err)
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

func (x *RepairAgent) attemptRepairMissingProposal(
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
		return nil, fmt.Errorf("no repaired proposal returned, proposal hash %s, vid %s",
			libxdr.HashString(missing.Proposal_hash), protoutil.VIDString(from))
	}

	// verify the returned proposal is the one committed to by the value
	if !protoutil.IsSaneProposal(reply) {
		return nil, fmt.Errorf("repaired proposal is not sane")
	}
	if !protoutil.ProposalMatchesValue(reply, missing) {
		return nil, fmt.Errorf("repaired proposal/value mismatch")
	}

	return &proto.ConsensusBasis{
		Value:        *missing,
		Proposal_txs: reply.Transactions,
	}, nil
}
