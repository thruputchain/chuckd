package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type ValueHash struct {
	bytes proto.SHA256Hash
}

func NewCVCHash(b []byte) ValueHash {
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
	return NewCVCHash(h.Sum(nil))
}

func ValueHashFromBasis(basis *proto.ConsensusBasis) ValueHash {
	return HashValue(BasisValue(basis))
}

// basis must be pre-validated with IsSaneConsensusBasis, otherwise panic.
func BasisValue(basis *proto.ConsensusBasis) *proto.ConsensusValue {
	return CVCGivenCVHash(basis, HashBasisFast(basis))
}

func CVCGivenCVHash(basis *proto.ConsensusBasis, basisHash proto.ConsensusBasisHash) *proto.ConsensusValue {
	return &proto.ConsensusValue{
		Proposer:        basis.Nomination.Proposer,
		Basis_hash:      basisHash,
		Prev_basis_hash: basis.Prev_basis_hash,
		Block_height:    basis.Nomination.Block_height,
		Block_hash:      basis.Nomination.Block_hash,
		Prev_block_hash: basis.Nomination.Prev_block_hash,
	}
}

func BasisAgreesWithValue(basis *proto.ConsensusBasis, cvc *proto.ConsensusValue) bool {
	return ValidatorInfoEquals(&cvc.Proposer, &basis.Nomination.Proposer) &&
		cvc.Block_hash == basis.Nomination.Block_hash &&
		cvc.Prev_block_hash == basis.Nomination.Prev_block_hash &&
		cvc.Prev_basis_hash == basis.Prev_basis_hash
}

func ValueEquals(v1, v2 *proto.ConsensusValue) bool {
	return *v1 == *v2
}

func ValidatorInfoEquals(v1, v2 *proto.ValidatorPublicInfo) bool {
	return *v1 == *v2
}
