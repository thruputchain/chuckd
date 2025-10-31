package repairagent

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	proto "github.com/groundhogchain/protocol-go"
)

func (x *RepairAgent) RepairMissingSCP(
	ctx context.Context,
	since scpkit.Slot,
) (*proto.ConsensusValue, error) {

	// XXX: sample source from discovery service

	panic("XXX")

}
