package protoutil

import (
	"crypto/sha256"
	"sync"

	"github.com/groundhogchain/chuckd/util"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func SimpleHashCV(m libxdr.XdrMarshaler, cv *proto.ConsensusValue) proto.ConsensusValueHash {
	bb := m.MarshalXdr(cv)
	defer bb.Recycle()
	return sha256.Sum256(bb.Bytes())
}

func HashCV(m libxdr.XdrMarshaler, cv *proto.ConsensusValue) proto.ConsensusValueHash {

	// XXX: len(Transaction_hashes) = len(Transaction_nominations), check in validate logic

	// cv minus txs
	cvz := *cv
	cvz.Nomination.Transaction_hashes = nil
	cvz.Nomination.Transaction_nominations = nil
	bb := m.MarshalXdr(&cvz)
	h0 := sha256.Sum256(bb.Bytes())
	bb.Recycle()

	// split txs into intervals
	const TxsPerThread = 25000
	n := len(cv.Nomination.Transaction_hashes)
	k := util.CeilDivSigned(n, TxsPerThread)
	hashes := make([][32]byte, k)

	// compute hashes of txs intervals
	var wg sync.WaitGroup
	wg.Add(k)
	for i := range k {
		go func() {
			defer wg.Done()
			lo := i * TxsPerThread
			hi := min(n, (i+1)*TxsPerThread)
			o := proto.NominationTxsOnly{
				Transaction_hashes:      cv.Nomination.Transaction_hashes[lo:hi],
				Transaction_nominations: cv.Nomination.Transaction_nominations[lo:hi],
			}
			bb := m.MarshalXdr(&o)
			defer bb.Recycle()
			hashes[i] = sha256.Sum256(bb.Bytes())
		}()
	}
	wg.Wait()

	// combine hashes of cv with txs intervals
	comb := sha256.New()
	comb.Write(h0[:])
	for _, h := range hashes {
		comb.Write(h[:])
	}

	var sum [32]byte
	comb.Sum(sum[:0])
	return sum
}

func IsSaneConsensusValue(cv *proto.ConsensusValue) bool {
	if len(cv.Nomination.Transaction_hashes) != len(cv.Nomination.Transaction_nominations) {
		return false
	}
	return true
}

func CueFromConsensusValue(m libxdr.XdrMarshaler, cv *proto.ConsensusValue) proto.ConsensusValueCue {
	return proto.ConsensusValueCue{
		Proposer:        cv.Nomination.Proposer,
		Cv_hash:         HashCV(m, cv),
		Prev_cv_hash:    cv.Prev_cv_hash,
		Block_height:    cv.Nomination.Block_height,
		Block_hash:      cv.Nomination.Block_hash,
		Prev_block_hash: cv.Nomination.Prev_block_hash,
	}
}
