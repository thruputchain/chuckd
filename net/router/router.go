package router

import (
	"context"

	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	proto "github.com/groundhogchain/protocol-go"
)

type ValidatorID = proto.ValidatorID
type ValidatorPublicInfo = proto.ValidatorPublicInfo

type Router interface {
	ChooseSCPProviders(ctx context.Context, seed int64) []ValidatorPublicInfo
	ChooseCatchupProviders(ctx context.Context, seed int64) []ValidatorPublicInfo
	Self() ValidatorPublicInfo
	OurQuorumSet() proto.SCPQuorumSet
	BuildDogwoodTree(rootVID ValidatorID, maxSize int) *dogwoodutil.LinkedTreeNode
	ResolveAddress(ValidatorID) (string, error)
}
