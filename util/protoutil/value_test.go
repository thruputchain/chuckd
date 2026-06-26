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

func TestConsensusValueCueRecursiveFieldEqualityMatchesGoValueEquality(t *testing.T) {
	base := sampleConsensusValue(11)

	testCases := []struct {
		name   string
		mutate func(*proto.ConsensusValue)
	}{
		{
			name:   "identical",
			mutate: func(*proto.ConsensusValue) {},
		},
		{
			name: "proposer public key differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposer.Public_key[0] ^= 0xFF
			},
		},
		{
			name: "proposer address differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Proposer.Address = "127.0.0.1:2002"
			},
		},
		{
			name: "cv hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Basis_hash[1] ^= 0xFF
			},
		},
		{
			name: "prev cv hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Prev_basis_hash[2] ^= 0xFF
			},
		},
		{
			name: "block height differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Block_height++
			},
		},
		{
			name: "block hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Block_hash[3] ^= 0xFF
			},
		},
		{
			name: "prev block hash differs",
			mutate: func(value *proto.ConsensusValue) {
				value.Prev_block_hash[4] ^= 0xFF
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			left := base
			right := base
			tc.mutate(&right)

			gotRecursiveFieldByField := consensusValueCueFieldByFieldEqual(left, right)
			gotDirectGoValue := left == right
			if gotRecursiveFieldByField != gotDirectGoValue {
				t.Fatalf(
					"recursive field-by-field comparison (%v) != direct Go value comparison (%v)",
					gotRecursiveFieldByField,
					gotDirectGoValue,
				)
			}

			gotCVCEquals := ValueEquals(&left, &right)
			if gotCVCEquals != gotDirectGoValue {
				t.Fatalf("CVCEquals (%v) != direct Go value comparison (%v)", gotCVCEquals, gotDirectGoValue)
			}
		})
	}
}

func consensusValueCueFieldByFieldEqual(left, right proto.ConsensusValue) bool {
	return left.Proposer.Public_key == right.Proposer.Public_key &&
		left.Proposer.Address == right.Proposer.Address &&
		left.Basis_hash == right.Basis_hash &&
		left.Prev_basis_hash == right.Prev_basis_hash &&
		left.Block_height == right.Block_height &&
		left.Block_hash == right.Block_hash &&
		left.Prev_block_hash == right.Prev_block_hash
}

func validatorPublicInfoFieldByFieldEqual(left, right proto.ValidatorPublicInfo) bool {
	return left.Public_key == right.Public_key &&
		left.Address == right.Address
}

func sampleConsensusValue(seed byte) proto.ConsensusValue {
	return proto.ConsensusValue{
		Proposer:        sampleValidatorPublicInfo(seed),
		Basis_hash:      filledConsensusBasisHash(seed + 1),
		Prev_basis_hash: filledConsensusBasisHash(seed + 2),
		Block_height:    42,
		Block_hash:      filledBlockHash(seed + 3),
		Prev_block_hash: filledBlockHash(seed + 4),
	}
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

func filledConsensusBasisHash(seed byte) proto.ConsensusBasisHash {
	var out proto.ConsensusBasisHash
	for i := range out {
		out[i] = seed + byte(i)
	}
	return out
}

func filledBlockHash(seed byte) proto.BlockHash {
	var out proto.BlockHash
	for i := range out {
		out[i] = seed + byte(i)
	}
	return out
}
