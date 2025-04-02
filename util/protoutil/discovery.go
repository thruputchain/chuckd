package protoutil

import (
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

func MakeValidatorQuorumSlicesSnapshot(
	privInfo proto.ValidatorPrivateInfo,
	slices []proto.QuorumSlice,
) proto.ValidatorQuorumSlicesSnapshot {
	qss := MakeQuorumsSlicesSnapshot(slices)
	return proto.ValidatorQuorumSlicesSnapshot{
		Validator_public_info:  privInfo.Public_info,
		Quorum_slices_snapshot: qss,
		Validator_signature:    SignXdr(privInfo.Secret_key, &qss),
	}
}

func MakeQuorumsSlicesSnapshot(slices []proto.QuorumSlice) proto.QuorumSlicesSnapshot {
	return proto.QuorumSlicesSnapshot{
		Timestamp:     time.Now().UnixNano(),
		Quorum_slices: slices,
	}
}

func VerifyValidatorQuorumSlicesSnapshot(
	value proto.ValidatorQuorumSlicesSnapshot,
) bool {
	return VerifyXdr(value.Validator_public_info.Public_key, &value.Quorum_slices_snapshot, value.Validator_signature)
}
