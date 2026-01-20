package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func HashCV(cv *proto.ConsensusValue) proto.ConsensusValueHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, cv)
	return proto.ConsensusValueHash(h.Sum(nil))
}

func IsSaneConsensusValue(cv *proto.ConsensusValue) bool {
	if xdrutil.SizeOfBatch(cv.Nomination.Transactions_hash) !=
		xdrutil.SizeOfBatch(cv.Nomination.Transactions_nondeterministic_data) {
		return false
	}
	return true
}

func CueFromConsensusValue(cv *proto.ConsensusValue) proto.ConsensusValueCue {
	return proto.ConsensusValueCue{
		Proposer:        cv.Nomination.Proposer,
		Cv_hash:         HashCV(cv),
		Prev_cv_hash:    cv.Prev_cv_hash,
		Block_height:    cv.Nomination.Block_height,
		Block_hash:      cv.Nomination.Block_hash,
		Prev_block_hash: cv.Nomination.Prev_block_hash,
	}
}
