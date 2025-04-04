package protoutil

import (
	"crypto/sha256"
	"fmt"

	proto "github.com/groundhogchain/protocol-go"
)

func ValidatorPublicKeyToID(pubKey proto.ValidatorPublicKey) proto.ValidatorID {
	return sha256.Sum256(pubKey[:])
}

func ValidatorPublicShortString(pubInfo proto.ValidatorPublicInfo) string {
	id := ValidatorPublicKeyToID(pubInfo.Public_key)
	return fmt.Sprintf("%s@%s", EncodeBase64(id[:]), pubInfo.Address)
}

func ValidatorIDEqual(vid1 proto.ValidatorID, vid2 proto.ValidatorID) bool {
	return vid1 == vid2
}
