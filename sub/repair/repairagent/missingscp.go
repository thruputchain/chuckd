package repairagent

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/sub/repair/repairkit"
	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

func (x *RepairAgent) RepairMissingSCP(
	ctx context.Context,
	since scpkit.Slot,
) (proto.SCPEnvelopes, error) {

	ps := x.router.ChooseSCPProviders(ctx, x.rand.Int63())
	if len(ps) == 0 {
		return nil, fmt.Errorf("no scp providers among known validators")
	}

	c, err := repairkit.DialRepairClient(ctx, x.net, ps[0].Address)
	if err != nil {
		return nil, fmt.Errorf("failed to dial repair client %v: %w", protoutil.VPublicShortString(ps[0]), err)
	}
	defer c.Close()

	envs, err := c.Repair_missing_scp_envelopes(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to repair missing scp envelopes for slot %d: %w", since, err)
	}

	return envs, nil
}
