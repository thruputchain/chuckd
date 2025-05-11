package baton

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libtime"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type Handler interface {
	libtime.Time
	BatchMempool
	TransactionMempool
	CommittedTransactionMempool
	Dissemination
	Consensus

	Signer() proto.ValidatorPublicInfo
	SignXdr(xdr.XdrType) proto.Signature
	VerifyXdr(signer proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool

	GetCatchupSources(context.Context) []proto.ValidatorPublicInfo

	RestoreDb(context.Context) (proto.BlockHeight, proto.BlockHash, proto.BlockID, error) // implemented by the coordinator service

	// XXX: 64-bit block ids are too small for preventing collisions with large block counts
	GenerateBlockID(context.Context) proto.BlockID
	GenerateID64(context.Context) ID64

	DialCatchup(context.Context, proto.ValidatorPublicInfo) (CatchupConn, error)

	DialDb(context.Context) (DbCoordConn, error)

	// DialExecs blocks and returns a set of executor the handler can spare for the next validation/proposal task.
	DialExecs(context.Context) ([]ExecCoordConn, error)

	PrioritizeTxs(context.Context, []*proto.SignedTransaction) []*proto.SignedTransaction
}

type ID64 int64

// BatchMempool provides batches of proposed transactions by other validators.
type BatchMempool interface {
	LookupBatch(context.Context, proto.BatchHash) ([]*proto.ProposedTransaction, bool)
}

// TransactionMempool provides all uncommitted transactions, our own and those of other validators.
type TransactionMempool interface {
	ListTxs(context.Context) []*proto.SignedTransaction
	RemoveTx(context.Context, proto.SignedTransactionHash)
}

// CommittedTransactionMempool keeps the hashes of committed transaction for some TTL.
type CommittedTransactionMempool interface {
	AddCommittedTx(context.Context, proto.SignedTransactionHash)
	IsCommittedTx(context.Context, proto.SignedTransactionHash) bool
}

type CatchupConn interface {
	proto.Catchup1
	Close() error
}

type DbCoordConn interface {
	proto.DbCoord1
	Close() error
}

type ExecCoordConn interface {
	proto.ExecutorCoord1
	Close() error
}

type Consensus interface {
	Nominate(ctx context.Context, slot int64, value xdr.XdrType, prev xdr.XdrType) bool
}

type Dissemination interface {
	Disseminate(ctx context.Context, value xdr.XdrType)
}
