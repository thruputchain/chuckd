package protoutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

const ShortHashLen = 5

func VIDOfPublic(v proto.ValidatorPublicInfo) proto.ValidatorID {
	return ValidatorPublicKeyToID(v.Public_key)
}

func ValidatorPublicKeyToID(pubKey proto.ValidatorPublicKey) proto.ValidatorID {
	return sha256.Sum256(pubKey[:])
}

func VIDString(vid proto.ValidatorID) string {
	return "vid:" + hex.EncodeToString(vid[:ShortHashLen])
}

func VPublicShortString(pubInfo proto.ValidatorPublicInfo) string {
	id := ValidatorPublicKeyToID(pubInfo.Public_key)
	return fmt.Sprintf("(%s address:%s)", VIDString(id), pubInfo.Address)
}

func ValidatorIDEqual(vid1 proto.ValidatorID, vid2 proto.ValidatorID) bool {
	return vid1 == vid2
}

// sorting

type ValidatorPublicInfos []proto.ValidatorPublicInfo

func (vs ValidatorPublicInfos) Len() int {
	return len(vs)
}

func (vs ValidatorPublicInfos) Less(i, j int) bool {
	return VIDString(VIDOfPublic(vs[i])) < VIDString(VIDOfPublic(vs[j]))
}

func (vs ValidatorPublicInfos) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs ValidatorPublicInfos) Sort() {
	sort.Sort(vs)
}
