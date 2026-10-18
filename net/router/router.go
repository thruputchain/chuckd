package router

import (
	"context"

	"github.com/thruputchain/chuckd/util/dogwoodutil"
	proto "github.com/thruputchain/thruput-protocol-go"
)

type ValidatorID = proto.ValidatorID
type ValidatorPublicInfo = proto.ValidatorPublicInfo

type Router interface {
	ChooseSCPProviders(ctx context.Context, seed int64) []ValidatorPublicInfo
	ChooseCatchupProviders(ctx context.Context, seed int64) []ValidatorPublicInfo
	Self() ValidatorPublicInfo
	OurQuorumSet() proto.SCPQuorumSet
	BuildDogwoodTree(rootVID ValidatorID, maxSize int) *dogwoodutil.LinkedDAGNode
	ResolveAddress(ValidatorID) (string, error)
}
