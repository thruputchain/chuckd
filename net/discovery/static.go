package discovery

import (
	"context"

	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
)

type StaticDiscovery struct {
	self    proto.ValidatorPrivateInfo
	ourQset proto.SCPQuorumSet
	known   map[proto.ValidatorID]proto.ValidatorPublicInfo
}

func NewStaticDiscovery(self proto.ValidatorPrivateInfo, known []proto.ValidatorPublicInfo) Discovery {
	k := map[proto.ValidatorID]proto.ValidatorPublicInfo{}
	// generate a qset, mostly to fulfill the discovery interface
	// this class is used only by standalone dogwood, which does not use qset
	qset := proto.SCPQuorumSet{Threshold: 1}
	for _, v := range known {
		k[protoutil.PublicVID(v)] = v
		qset.Validators = append(qset.Validators, protoutil.ValidatorToNodeID(protoutil.PublicVID(v)))
	}
	return &StaticDiscovery{self: self, ourQset: qset, known: k}
}

func (x *StaticDiscovery) OurQuorumSet() proto.SCPQuorumSet {
	return x.ourQset
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

func (x *StaticDiscovery) GetQuorumSet(ctx context.Context, hash proto.SCPHash) (*proto.SCPQuorumSet, error) {
	panic("not used")
}
