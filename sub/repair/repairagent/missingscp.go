package repairagent

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	proto "github.com/groundhogchain/protocol-go"
)

func (x *RepairAgent) RepairMissingSCP(
	ctx context.Context,
	since scpkit.Slot,
) (proto.SCPEnvelopes, error) {

	// XXX: sample source from router service
	// x.router.ChooseSCPProviders(ctx, XXX)

	panic("XXX")

}
