package protoutil

import (
	"crypto/sha256"
	"sync"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func HashBasisSlow(basis *proto.ConsensusBasis) proto.ConsensusBasisHash {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, basis)
	return proto.ConsensusBasisHash(h.Sum(nil))
}

func HashBasisFast(basis *proto.ConsensusBasis) proto.ConsensusBasisHash {
	cvBase := *basis
	cvBase.Nomination.Transactions_hash = proto.ParallelBatch{}
	cvBase.Nomination.Transactions_nondeterministic_data = proto.ParallelBatch{}

	var hashBase proto.SHA256Hash
	var hashTxsHash proto.SHA256Hash
	var hashTxsND proto.SHA256Hash

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		hashBase = HashBasisSlow(&cvBase)
	}()
	go func() {
		defer wg.Done()
		hashTxsHash = xdrutil.HashParallelBatch(basis.Nomination.Transactions_hash)
	}()
	go func() {
		defer wg.Done()
		hashTxsND = xdrutil.HashParallelBatch(basis.Nomination.Transactions_nondeterministic_data)
	}()
	wg.Wait()
	for i := range hashBase {
		hashBase[i] ^= hashTxsHash[i] ^ hashTxsND[i]
	}
	return hashBase
}

func IsSaneConsensusBasis(basis *proto.ConsensusBasis) bool {
	if err := xdrutil.ValidateBatch(basis.Nomination.Transactions_hash); err != nil {
		return false
	}
	if err := xdrutil.ValidateBatch(basis.Nomination.Transactions_nondeterministic_data); err != nil {
		return false
	}
	if xdrutil.TokenSizeOfBatch(basis.Nomination.Transactions_hash) !=
		xdrutil.TokenSizeOfBatch(basis.Nomination.Transactions_nondeterministic_data) {
		return false
	}
	return true
}
