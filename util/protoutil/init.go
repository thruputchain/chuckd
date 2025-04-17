package protoutil

import (
	"context"
	"crypto/ed25519"
	"math/rand"

	"github.com/groundhogchain/chuckd/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

func InitConfig(ctx context.Context, rr *rand.Rand, address string) *proto.ValidatorConfig {
	privInfo := MustGenerateIdentity(ctx, rr, address)
	return &proto.ValidatorConfig{Private_info: *privInfo}
}

func MustGenerateIdentity(ctx context.Context, rr *rand.Rand, address string) *proto.ValidatorPrivateInfo {
	p, err := GenerateIdentity(rr, address)
	must.NoError(ctx, err)
	return p
}

func GenerateIdentity(rr *rand.Rand, address string) (*proto.ValidatorPrivateInfo, error) {
	pubKey, secretKey, err := ed25519.GenerateKey(rr)
	if err != nil {
		return nil, err
	}
	return &proto.ValidatorPrivateInfo{
		Public_info: proto.ValidatorPublicInfo{
			Public_key: proto.ValidatorPublicKey(pubKey),
			Address:    address,
		},
		Secret_key: proto.ValidatorSecretKey(secretKey),
	}, nil
}
