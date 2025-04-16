package protoutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	proto "github.com/groundhogchain/protocol-go"
)

func ValidatorIDFromPublic(v proto.ValidatorPublicInfo) proto.ValidatorID {
	return ValidatorPublicKeyToID(v.Public_key)
}

func ValidatorPublicKeyToID(pubKey proto.ValidatorPublicKey) proto.ValidatorID {
	return sha256.Sum256(pubKey[:])
}

func ValidatorIDString(vid proto.ValidatorID) string {
	return "vid:" + hex.EncodeToString(vid[:])
}

func ValidatorPublicShortString(pubInfo proto.ValidatorPublicInfo) string {
	id := ValidatorPublicKeyToID(pubInfo.Public_key)
	return fmt.Sprintf("(%s address:%s)", ValidatorIDString(id), pubInfo.Address)
}

func ValidatorIDEqual(vid1 proto.ValidatorID, vid2 proto.ValidatorID) bool {
	return vid1 == vid2
}
