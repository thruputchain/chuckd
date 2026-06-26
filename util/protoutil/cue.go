package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type CVCHash struct {
	bytes proto.SHA256Hash
}

func NewCVCHash(b []byte) CVCHash {
	return CVCHash{bytes: proto.SHA256Hash(b)}
}

func (x CVCHash) Bytes() proto.SHA256Hash {
	return x.bytes
}

func (x CVCHash) Slice() []byte {
	return x.bytes[:]
}

func (x CVCHash) Display() string {
	return libxdr.HashString(x.Bytes())
}

func HashCVC(cvc *proto.ConsensusValueCue) CVCHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, cvc)
	return NewCVCHash(h.Sum(nil))
}

func CVCHashFromBasis(basis *proto.ConsensusBasis) CVCHash {
	return HashCVC(CVC(basis))
}

// basis must be pre-validated with IsSaneConsensusBasis, otherwise panic.
func CVC(basis *proto.ConsensusBasis) *proto.ConsensusValueCue {
	return CVCGivenCVHash(basis, HashBasisFast(basis))
}

func CVCGivenCVHash(basis *proto.ConsensusBasis, basisHash proto.ConsensusBasisHash) *proto.ConsensusValueCue {
	return &proto.ConsensusValueCue{
		Proposer:        basis.Nomination.Proposer,
		Cv_hash:         basisHash,
		Prev_cv_hash:    basis.Prev_cv_hash,
		Block_height:    basis.Nomination.Block_height,
		Block_hash:      basis.Nomination.Block_hash,
		Prev_block_hash: basis.Nomination.Prev_block_hash,
	}
}

func BasisAgreesWithCue(basis *proto.ConsensusBasis, cvc *proto.ConsensusValueCue) bool {
	return ValidatorPublicInfoEquals(&cvc.Proposer, &basis.Nomination.Proposer) &&
		cvc.Block_hash == basis.Nomination.Block_hash &&
		cvc.Prev_block_hash == basis.Nomination.Prev_block_hash &&
		cvc.Prev_cv_hash == basis.Prev_cv_hash
}

func CVCEquals(cvc1, cvc2 *proto.ConsensusValueCue) bool {
	return *cvc1 == *cvc2
}

func ValidatorPublicInfoEquals(v1, v2 *proto.ValidatorPublicInfo) bool {
	return *v1 == *v2
}
