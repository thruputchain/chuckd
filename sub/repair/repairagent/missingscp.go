package repairagent

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/sub/repair/repairkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	proto "github.com/groundhogchain/protocol-go"
)

func (x *RepairAgent) RepairMissingSCP(
	ctx context.Context,
	since scpkit.Slot,
) (proto.SCPEnvelopes, error) {

	ps := x.router.ChooseSCPProviders(ctx, x.rand.Int63())
	if len(ps) == 0 {
		return nil, fmt.Errorf("no scp providers in quorum set")
	}

	c, err := repairkit.DialRepairClient(ctx, x.net, ps[0].Address)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	envs, err := c.Repair_missing_scp_envelopes(ctx, since)
	if err != nil {
		return nil, err
	}

	return envs, nil
}
