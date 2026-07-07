package protoutil

import (
	"testing"

	proto "github.com/groundhogchain/protocol-go"
)

func TestValidatorPublicInfoRecursiveFieldEqualityMatchesGoValueEquality(t *testing.T) {
	base := sampleValidatorPublicInfo(41)

	testCases := []struct {
		name   string
		mutate func(*proto.ValidatorPublicInfo)
	}{
		{
			name:   "identical",
			mutate: func(*proto.ValidatorPublicInfo) {},
		},
		{
			name: "public key differs",
			mutate: func(info *proto.ValidatorPublicInfo) {
				info.Public_key[0] ^= 0xFF
			},
		},
		{
			name: "address differs",
			mutate: func(info *proto.ValidatorPublicInfo) {
				info.Address = "127.0.0.1:9002"
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			left := base
			right := base
			tc.mutate(&right)

			gotRecursiveFieldByField := validatorPublicInfoFieldByFieldEqual(left, right)
			gotDirectGoValue := left == right
			if gotRecursiveFieldByField != gotDirectGoValue {
				t.Fatalf(
					"recursive field-by-field comparison (%v) != direct Go value comparison (%v)",
					gotRecursiveFieldByField,
					gotDirectGoValue,
				)
			}

			gotValidatorPublicInfoEquals := ValidatorInfoEquals(&left, &right)
			if gotValidatorPublicInfoEquals != gotDirectGoValue {
				t.Fatalf(
					"ValidatorPublicInfoEquals (%v) != direct Go value comparison (%v)",
					gotValidatorPublicInfoEquals,
					gotDirectGoValue,
				)
			}
		})
	}
}

func TestValueEqualsDetectsFieldDifferences(t *testing.T) {
	base := sampleConsensusValue(11)

	testCases := []struct {
		name      string
		mutate    func(*proto.ConsensusValue)
		wantEqual bool
	}{
		{
			name:      "identical",
			mutate:    func(*proto.ConsensusValue) {},
			wantEqual: true,
		},
		{
			name: "prev value hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Prev_value_hash[2] ^= 0xFF
			},
		},
		{
			name: "block timepoint differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposal_header.Block_timepoint++
			},
		},
		{
			name: "proposer differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposal_header.Proposer[0] ^= 0xFF
			},
		},
		{
			name: "upgrades differ",
			mutate: func(value *proto.ConsensusValue) {
				value.Upgrades = []proto.Upgrade{{0xFF}}
			},
		},
		{
			name: "network hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposal_header.Network_hash[1] ^= 0xFF
			},
		},
		{
			name: "block height differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposal_header.Block_height++
			},
		},
		{
			name: "txs hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposal_header.Txs_hash[3] ^= 0xFF
			},
		},
		{
			name: "storage hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposal_header.Storage_hash[4] ^= 0xFF
			},
		},
		{
			name: "events hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposal_header.Events_hash[5] ^= 0xFF
			},
		},
		{
			name: "proposal txs hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposal_txs_hash[6] ^= 0xFF
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			left := base
			right := base
			tc.mutate(&right)

			if got := ValueEquals(&left, &right); got != tc.wantEqual {
				t.Fatalf("ValueEquals = %v, want %v", got, tc.wantEqual)
			}
		})
	}
}

func validatorPublicInfoFieldByFieldEqual(left, right proto.ValidatorPublicInfo) bool {
	return left.Public_key == right.Public_key &&
		left.Address == right.Address
}

func sampleConsensusValue(seed byte) proto.ConsensusValue {
	return proto.ConsensusValue{
		Prev_value_hash: filledHash(seed + 2),
		Proposal_header: proto.ProposalHeader{
			Network_hash:    filledHash(seed + 1),
			Proposer:        sampleValidatorID(seed),
			Block_timepoint: proto.TimePoint(seed) + 100,
			Block_height:    42,
			Txs_hash:        filledHash(seed + 3),
			Storage_hash:    filledHash(seed + 4),
			Events_hash:     filledHash(seed + 5),
		},
		Proposal_txs_hash: filledHash(seed + 6),
		Upgrades:          sampleUpgrades(seed),
	}
}

func sampleUpgrades(seed byte) []proto.Upgrade {
	return []proto.Upgrade{
		{seed, seed + 1, seed + 2},
		{seed + 3, seed + 4},
	}
}

func sampleValidatorID(seed byte) proto.ValidatorID {
	var out proto.ValidatorID
	for i := range out {
		out[i] = seed + byte(i)
	}
	return out
}

func sampleValidatorPublicInfo(seed byte) proto.ValidatorPublicInfo {
	return proto.ValidatorPublicInfo{
		Public_key: filledPublicKey(seed),
		Address:    "127.0.0.1:2001",
	}
}

func filledPublicKey(seed byte) proto.ValidatorPublicKey {
	var out proto.ValidatorPublicKey
	for i := range out {
		out[i] = seed + byte(i)
	}
	return out
}

func filledHash(seed byte) proto.SHA256Hash {
	var out proto.SHA256Hash
	for i := range out {
		out[i] = seed + byte(i)
	}
	return out
}
