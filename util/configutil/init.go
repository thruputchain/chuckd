package configutil

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func InitConfig(ctx context.Context) *proto.Config {
	privInfo, err := GenerateCredentials()
	must.NoError(ctx, err)
	return &proto.Config{Private_info: *privInfo}
}

func GenerateCredentials() (*proto.ValidatorPrivateInfo, error) {
	pubKey, secretKey, err := ed25519.GenerateKey(nil) // for safety, uses non-deterministic randomness from crypto package
	if err != nil {
		return nil, err
	}
	return &proto.ValidatorPrivateInfo{
		Public_info: proto.ValidatorPublicInfo{
			Public_key: proto.ValidatorPublicKey(pubKey),
		},
		Secret_key: proto.ValidatorSecretKey(secretKey),
	}, nil
}

func Ed25519PubKeyToID(pubKey proto.ValidatorPublicKey) proto.ValidatorID {
	return sha256.Sum256(pubKey[:])
}
