package protoutil

import (
	"bytes"
	"crypto/sha256"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type ValueHash struct {
	bytes proto.SHA256Hash
}

func NewValueHash(b []byte) ValueHash {
	return ValueHash{bytes: proto.SHA256Hash(b)}
}

func (x ValueHash) Bytes() proto.SHA256Hash {
	return x.bytes
}

func (x ValueHash) Slice() []byte {
	return x.bytes[:]
}

func (x ValueHash) Display() string {
	return libxdr.HashString(x.Bytes())
}

func HashValue(value *proto.ConsensusValue) ValueHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, value)
	return NewValueHash(h.Sum(nil))
}

func ValueHashFromBasis(basis *proto.ConsensusBasis) ValueHash {
	return HashValue(BasisValue(basis))
}

// basis must be pre-validated with IsSaneConsensusBasis, otherwise panic.
func BasisValue(basis *proto.ConsensusBasis) *proto.ConsensusValue {
	return ValueGivenBasisHash(basis, HashBasisFast(basis))
}

func ValueGivenBasisHash(basis *proto.ConsensusBasis, basisHash proto.ConsensusBasisHash) *proto.ConsensusValue {
	return &proto.ConsensusValue{
		Timestamp:       basis.Timestamp,
		Proposer:        basis.Proposer,
		Upgrades:        basis.Upgrades,
		Basis_hash:      basisHash,
		Prev_basis_hash: basis.Prev_basis_hash,
		Block_height:    basis.Nomination.Block_height,
		Block_hash:      basis.Nomination.Block_hash,
		Prev_block_hash: basis.Nomination.Prev_block_hash,
		Prev_value:      basis.Prev_value,
	}
}

func BasisAgreesWithValue(basis *proto.ConsensusBasis, value *proto.ConsensusValue) bool {
	return value.Timestamp == basis.Timestamp &&
		ValidatorIDEqual(value.Proposer, basis.Proposer) &&
		upgradesEqual(value.Upgrades, basis.Upgrades) &&
		value.Block_hash == basis.Nomination.Block_hash &&
		value.Prev_block_hash == basis.Nomination.Prev_block_hash &&
		value.Prev_basis_hash == basis.Prev_basis_hash &&
		value.Prev_value == basis.Prev_value
}

// ValueEquals reports whether two ConsensusValues are equal. It compares
// field by field because ConsensusValue contains an Upgrades slice and is
// therefore not comparable with the == operator.
func ValueEquals(v1, v2 *proto.ConsensusValue) bool {
	return v1.Timestamp == v2.Timestamp &&
		v1.Proposer == v2.Proposer &&
		v1.Block_height == v2.Block_height &&
		v1.Prev_value == v2.Prev_value &&
		v1.Basis_hash == v2.Basis_hash &&
		v1.Prev_basis_hash == v2.Prev_basis_hash &&
		v1.Block_hash == v2.Block_hash &&
		v1.Prev_block_hash == v2.Prev_block_hash &&
		upgradesEqual(v1.Upgrades, v2.Upgrades)
}

func upgradesEqual(a, b []proto.Upgrade) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func ValidatorInfoEquals(v1, v2 *proto.ValidatorPublicInfo) bool {
	return *v1 == *v2
}
