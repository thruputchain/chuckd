package discovery

import (
	"context"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type TestDiscovery struct {
	selfIndex int
	vs        []*proto.ValidatorPrivateInfo
}

func NewTestDiscovery(ctx context.Context, vs []*proto.ValidatorPrivateInfo, selfIndex int) *TestDiscovery {
	return &TestDiscovery{
		selfIndex: selfIndex,
		vs:        vs,
	}
}

func (x *TestDiscovery) SelfID() proto.ValidatorID {
	return protoutil.VIDOfPublic(x.vs[x.selfIndex].Public_info)
}

func (x *TestDiscovery) SecretKey() proto.ValidatorSecretKey {
	return x.vs[x.selfIndex].Secret_key
}

func (x *TestDiscovery) LookupValidator(vid proto.ValidatorID) (proto.ValidatorPublicInfo, bool) {
	for _, v := range x.vs {
		if vid == protoutil.VIDOfPublic(v.Public_info) {
			return v.Public_info, true
		}
	}
	return proto.ValidatorPublicInfo{}, false
}

func (x *TestDiscovery) LookupTrustedValidator(vid proto.ValidatorID) (proto.ValidatorPublicInfo, bool) {
	return x.LookupValidator(vid)
}

func (x *TestDiscovery) CachedSnapshots() proto.KnownQuorumSets {
	panic("not used")
	// kqs := proto.KnownQuorumSets{
	// 	Quorum_set_snapshots: make([]proto.SignedQuorumSetSnapshot, len(x.vs)),
	// 	Validator_snapshots:  make([]proto.SignedValidatorInfoSnapshot, len(x.vs)),
	// }
	// for i := range kqs.Quorum_set_snapshots {
	// 	kqs.Quorum_set_snapshots[i] = proto.SignedQuorumSetSnapshot{}
	// }
	// return kqs
}
