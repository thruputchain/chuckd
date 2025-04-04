package protoutil

import proto "github.com/groundhogchain/protocol-go"

func SnapshotsValidatorSet(vqss proto.ValidatorQuorumSlicesSnapshots) []proto.ValidatorPublicInfo {
	vs := []proto.ValidatorPublicInfo{}
	for _, vqs := range vqss {
		for _, vq := range vqs.Quorum_slices_snapshot.Quorum_slices {
			vs = append(vs, vq...)
		}
	}
	return vs
}
