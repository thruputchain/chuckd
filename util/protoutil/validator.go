package protoutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
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

func EncodeBase64(v []byte) string {
	var w bytes.Buffer
	enc := base64.NewEncoder(base64.StdEncoding, &w)
	enc.Write(v)
	enc.Close()
	return w.String()
}
