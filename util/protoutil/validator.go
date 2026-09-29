package protoutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

const ShortHashLen = 3

func ToSecretKey(v proto.ValidatorSecretKey) proto.SecretKey {
	return v
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

// NodeToValidatorID requires an Ed25519 key. Validate network keys before calling it.
func NodeToValidatorID(pk proto.SCPNodeID) proto.ValidatorID {
	return *pk.Ed25519()
}

func PublicVID(v proto.ValidatorPublicInfo) proto.ValidatorID {
	return VPublicKeyToID(v.Public_key)
}

// VPublicKeyToID requires an Ed25519 key. Validate network keys before calling it.
func VPublicKeyToID(pubKey proto.ValidatorPublicKey) proto.ValidatorID {
	return *pubKey.Ed25519()
}

func VIDString(vid proto.ValidatorID) string {
	return "vid:" + hex.EncodeToString(vid[:ShortHashLen])
}

func VIDUntypedString(vid proto.ValidatorID) string {
	return hex.EncodeToString(vid[:ShortHashLen])
}

func VPublicDisplay(pubInfo proto.ValidatorPublicInfo) string {
	if pubInfo.Public_key.Scheme != proto.Ed25519 {
		return fmt.Sprintf("(unsupported public key scheme:%d address:%s)", pubInfo.Public_key.Scheme, pubInfo.Address)
	}
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
	vi := PublicVID(vs[i])
	vj := PublicVID(vs[j])
	return bytes.Compare(vi[:], vj[:]) < 0
}

func (vs ValidatorPublicInfos) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs ValidatorPublicInfos) Sort() {
	sort.Sort(vs)
}
