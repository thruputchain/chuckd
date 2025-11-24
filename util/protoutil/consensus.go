package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func HashCV(m xdrutil.XdrMarshaler, cv *proto.ConsensusValue) proto.ConsensusValueHash {
	bb := m.MarshalXdr(cv)
	defer bb.Recycle()
	return sha256.Sum256(bb.Bytes())
}

func IsSaneConsensusValue(cv *proto.ConsensusValue) bool {
	if len(cv.Nomination.Transaction_hashes) != len(cv.Nomination.Transaction_nominations) {
		return false
	}
	return true
}

func CueFromConsensusValue(m xdrutil.XdrMarshaler, cv *proto.ConsensusValue) proto.ConsensusValueCue {
	return proto.ConsensusValueCue{
		Proposer:        cv.Nomination.Proposer,
		Cv_hash:         HashCV(m, cv),
		Prev_cv_hash:    cv.Prev_cv_hash,
		Block_height:    cv.Nomination.Block_height,
		Block_hash:      cv.Nomination.Block_hash,
		Prev_block_hash: cv.Nomination.Prev_block_hash,
	}
}
