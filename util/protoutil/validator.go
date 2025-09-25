package protoutil

import (
	"encoding/hex"
	"fmt"
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

const ShortHashLen = 5

func ToSecretKey(v proto.ValidatorSecretKey) proto.SecretKey {
	sk := proto.SecretKey{}
	*sk.Ed25519() = proto.Ed25519SecretKey(v)
	return sk
}

func ValidatorToNodeIDs(vids []proto.ValidatorID) []proto.SCPNodeID {
	ns := make([]proto.SCPNodeID, len(vids))
	for i, vid := range vids {
		ns[i] = ValidatorToNodeID(vid)
	}
	return ns
}

func ValidatorToNodeID(vid proto.ValidatorID) proto.SCPNodeID {
	pk := proto.PublicKey{}
	pk.Scheme = proto.Ed25519
	*pk.Ed25519() = proto.Ed25519PublicKey(vid)
	return pk
}

func PublicVID(v proto.ValidatorPublicInfo) proto.ValidatorID {
	return VPublicKeyToID(v.Public_key)
}

func VPublicKeyToID(pubKey proto.ValidatorPublicKey) proto.ValidatorID {
	return proto.ValidatorID(pubKey)
}

func VIDString(vid proto.ValidatorID) string {
	return "vid:" + hex.EncodeToString(vid[:ShortHashLen])
}

func VPublicShortString(pubInfo proto.ValidatorPublicInfo) string {
	id := VPublicKeyToID(pubInfo.Public_key)
	return fmt.Sprintf("(%s address:%s)", VIDString(id), pubInfo.Address)
}

func ValidatorIDEqual(vid1 proto.ValidatorID, vid2 proto.ValidatorID) bool {
	return vid1 == vid2
}

func ValidatorPublicToIDs(vs []proto.ValidatorPublicInfo) map[proto.ValidatorID]bool {
	ids := map[proto.ValidatorID]bool{}
	for _, v := range vs {
		ids[PublicVID(v)] = true
	}
	return ids
}

// sorting

type ValidatorPublicInfos []proto.ValidatorPublicInfo

func (vs ValidatorPublicInfos) Len() int {
	return len(vs)
}

func (vs ValidatorPublicInfos) Less(i, j int) bool {
	return VIDString(PublicVID(vs[i])) < VIDString(PublicVID(vs[j]))
}

func (vs ValidatorPublicInfos) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs ValidatorPublicInfos) Sort() {
	sort.Sort(vs)
}
