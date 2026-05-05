package protoutil

import (
	"crypto/sha256"
	"sync"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func HashCVSlow(cv *proto.ConsensusValue) proto.ConsensusValueHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, cv)
	return proto.ConsensusValueHash(h.Sum(nil))
}

func HashCVFast(cv *proto.ConsensusValue) proto.ConsensusValueHash {
	cvBase := *cv
	cvBase.Nomination.Transactions_hash = proto.ParallelBatch{}
	cvBase.Nomination.Transactions_nondeterministic_data = proto.ParallelBatch{}

	var hashBase proto.SHA256Hash
	var hashTxsHash proto.SHA256Hash
	var hashTxsND proto.SHA256Hash

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		hashBase = HashCVSlow(&cvBase)
	}()
	go func() {
		defer wg.Done()
		hashTxsHash = xdrutil.HashParallelBatch(cv.Nomination.Transactions_hash)
	}()
	go func() {
		defer wg.Done()
		hashTxsND = xdrutil.HashParallelBatch(cv.Nomination.Transactions_nondeterministic_data)
	}()
	wg.Wait()
	for i := range hashBase {
		hashBase[i] ^= hashTxsHash[i] ^ hashTxsND[i]
	}
	return hashBase
}

func IsSaneConsensusValue(cv *proto.ConsensusValue) bool {
	if err := xdrutil.ValidateBatch(cv.Nomination.Transactions_hash); err != nil {
		return false
	}
	if err := xdrutil.ValidateBatch(cv.Nomination.Transactions_nondeterministic_data); err != nil {
		return false
	}
	if xdrutil.TokenSizeOfBatch(cv.Nomination.Transactions_hash) !=
		xdrutil.TokenSizeOfBatch(cv.Nomination.Transactions_nondeterministic_data) {
		return false
	}
	return true
}

type CVCHash proto.SHA256Hash

func HashCVC(cvc *proto.ConsensusValueCue) CVCHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, cvc)
	return CVCHash(h.Sum(nil))
}

// cv must be pre-validated with IsSaneConsensusValue, otherwise panic.
func CVC(cv *proto.ConsensusValue) proto.ConsensusValueCue {
	return CVCGivenCVHash(cv, HashCVFast(cv))
}

func CVCGivenCVHash(cv *proto.ConsensusValue, cvhash proto.ConsensusValueHash) proto.ConsensusValueCue {
	return proto.ConsensusValueCue{
		Proposer:        cv.Nomination.Proposer,
		Cv_hash:         cvhash,
		Prev_cv_hash:    cv.Prev_cv_hash,
		Block_height:    cv.Nomination.Block_height,
		Block_hash:      cv.Nomination.Block_hash,
		Prev_block_hash: cv.Nomination.Prev_block_hash,
	}
}

func CVAgreesWithCue(cv *proto.ConsensusValue, cvc *proto.ConsensusValueCue) bool {
	//XXX: check proposer matches
	return cvc.Block_hash == cv.Nomination.Block_hash &&
		cvc.Prev_block_hash == cv.Nomination.Prev_block_hash &&
		cvc.Prev_cv_hash == cv.Prev_cv_hash
}

func CVCEquals(cvc1, cvc2 proto.ConsensusValueCue) bool {
	// XXX: cvc1.Proposer == cvc2.Proposer
	return cvc1.Block_height == cvc2.Block_height &&
		cvc1.Block_hash == cvc2.Block_hash &&
		cvc1.Prev_block_hash == cvc2.Prev_block_hash &&
		cvc1.Prev_cv_hash == cvc2.Prev_cv_hash
}
