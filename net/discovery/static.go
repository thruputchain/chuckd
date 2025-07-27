package discovery

import (
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type StaticDiscovery struct {
	self  proto.ValidatorPrivateInfo
	known map[proto.ValidatorID]proto.ValidatorPublicInfo
}

func NewStaticDiscovery(self proto.ValidatorPrivateInfo, known []proto.ValidatorPublicInfo) Discovery {
	k := map[proto.ValidatorID]proto.ValidatorPublicInfo{}
	for _, v := range known {
		k[protoutil.PublicVID(v)] = v
	}
	return &StaticDiscovery{self: self, known: k}
}

func (x *StaticDiscovery) SelfID() proto.ValidatorID {
	return protoutil.PublicVID(x.self.Public_info)
}

func (x *StaticDiscovery) SecretKey() proto.ValidatorSecretKey {
	return x.self.Secret_key
}

func (x *StaticDiscovery) LookupValidator(vid proto.ValidatorID) (proto.ValidatorPublicInfo, bool) {
	v, ok := x.known[vid]
	return v, ok
}

func (x *StaticDiscovery) LookupTrustedValidator(vid proto.ValidatorID) (proto.ValidatorPublicInfo, bool) {
	return x.LookupValidator(vid)
}

func (x *StaticDiscovery) CachedSnapshots() proto.KnownQuorumSets {
	panic("not used")
}
