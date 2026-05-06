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

func CVCHashFromCV(cv *proto.ConsensusValue) CVCHash {
	return HashCVC(CVC(cv))
}

// cv must be pre-validated with IsSaneConsensusValue, otherwise panic.
func CVC(cv *proto.ConsensusValue) *proto.ConsensusValueCue {
	return CVCGivenCVHash(cv, HashCVFast(cv))
}

func CVCGivenCVHash(cv *proto.ConsensusValue, cvhash proto.ConsensusValueHash) *proto.ConsensusValueCue {
	return &proto.ConsensusValueCue{
		Proposer:        cv.Nomination.Proposer,
		Cv_hash:         cvhash,
		Prev_cv_hash:    cv.Prev_cv_hash,
		Block_height:    cv.Nomination.Block_height,
		Block_hash:      cv.Nomination.Block_hash,
		Prev_block_hash: cv.Nomination.Prev_block_hash,
	}
}

func CVAgreesWithCue(cv *proto.ConsensusValue, cvc *proto.ConsensusValueCue) bool {
	return cvc.Block_hash == cv.Nomination.Block_hash &&
		cvc.Prev_block_hash == cv.Nomination.Prev_block_hash &&
		cvc.Prev_cv_hash == cv.Prev_cv_hash
}

func CVCEquals(cvc1, cvc2 *proto.ConsensusValueCue) bool {
	return cvc1.Block_height == cvc2.Block_height &&
		cvc1.Block_hash == cvc2.Block_hash &&
		cvc1.Prev_block_hash == cvc2.Prev_block_hash &&
		cvc1.Prev_cv_hash == cvc2.Prev_cv_hash
}
