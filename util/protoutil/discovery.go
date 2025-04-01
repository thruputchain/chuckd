package protoutil

import (
	proto "github.com/groundhogchain/protocol-go"
)

func Sign_ValidatorQuorumSlicesSnapshot(
	secretKey proto.ValidatorSecretKey,
	value *proto.ValidatorQuorumSlicesSnapshot,
) {
	value.Validator_signature = SignXdr(secretKey, &value.Quorum_slices_snapshot)
}

func Verify_ValidatorQuorumSlicesSnapshot(
	value proto.ValidatorQuorumSlicesSnapshot,
) bool {
	return VerifyXdr(value.Validator_public_info.Public_key, &value.Quorum_slices_snapshot, value.Validator_signature)
}
