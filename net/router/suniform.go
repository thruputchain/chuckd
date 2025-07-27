package router

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

type staticUniformRouter struct {
	m          xdrutil.XdrMarshaler
	maxFanOut  int
	self       proto.ValidatorPublicInfo
	validators []proto.ValidatorPublicInfo
}

func NewStaticUniformRouter(
	ctx context.Context,
	self proto.ValidatorPublicInfo,
	vs []proto.ValidatorPublicInfo,
	maxFanOut int,
) Router {
	return &staticUniformRouter{
		m:          xdrutil.NewXdrMarshaler(),
		maxFanOut:  maxFanOut,
		self:       self,
		validators: vs,
	}
}

func (x *staticUniformRouter) Self() proto.ValidatorPublicInfo {
	return x.self
}

func (x *staticUniformRouter) ResolveAddress(vid proto.ValidatorID) (string, error) {
	if vid == protoutil.PublicVID(x.self) {
		return x.self.Address, nil
	}
	for _, k := range x.validators {
		if vid == protoutil.PublicVID(k) {
			return k.Address, nil
		}
	}
	return "", fmt.Errorf("validator not found")
}

func (x *staticUniformRouter) ChooseCatchupProviders(ctx context.Context, seed int64) []proto.ValidatorPublicInfo {
	vs := append([]ValidatorPublicInfo{x.self}, x.validators...)
	_, vs = dedupFilterOutValidators(vs, protoutil.PublicVID(x.Self())) // filter out self
	must.Assertf(ctx, len(vs) > 0, "empty quorum")
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(vs), func(i int, j int) { vs[i], vs[j] = vs[j], vs[i] })
	return vs[:1]
}

func (x *staticUniformRouter) BuildDogwoodTree(rootID ValidatorID) *dogwoodutil.LinkedTreeNode {
	vs := protoutil.ValidatorPublicToIDs(x.validators)
	return buildDogwoodTree(x.maxFanOut, rootID, vs)
}

func (x *staticUniformRouter) BuildTree(rootID ValidatorID, cid ChannelID) (
	treeNodes MulticastTreeNodes,
	children []ValidatorPublicInfo,
	treeHash MulticastTreeHash,
) {
	vs := append([]ValidatorPublicInfo{x.self}, x.validators...)
	return buildTree(x.m, x.maxFanOut, vs, rootID, cid)
}
