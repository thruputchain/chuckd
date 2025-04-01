package protoutil

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
	// For safety, use non-deterministic randomness from crypto package, instead of
	// seeding the randomness source with the runtime seed.
	// Alternatively, we could force the user to provide a seed on the command-line,
	// when generating credentials.
	pubKey, secretKey, err := ed25519.GenerateKey(nil)
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

func ValidatorPublicKeyToID(pubKey proto.ValidatorPublicKey) proto.ValidatorID {
	return sha256.Sum256(pubKey[:])
}
