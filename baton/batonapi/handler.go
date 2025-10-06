package batonapi

import (
	"context"
	"fmt"

	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/protoutil"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type OpsHandler interface {
	SubmittedTxMempool
	CommittedTxMempool
	DialDb(context.Context) (DbCoordConn, error)
	DialExecs(context.Context) ([]ExecCoordConn, error)
	GenerateBlockID(context.Context) proto.BlockID
	Committed(context.Context, *proto.CatchupBlock)
	Emit(ctx context.Context, event any)
}

type Handler interface {
	protoutil.Signatory
	libtime.Time
	SubmittedTxMempool
	CommittedTxMempool
	Consensus

	GetCatchupSources(context.Context, int64) []proto.ValidatorPublicInfo

	RestoreDb(context.Context) (Tip, error) // implemented by the coordinator service

	GenerateBlockID(context.Context) proto.BlockID
	GenerateID64(context.Context) ID64

	DialCatchup(context.Context, proto.ValidatorPublicInfo) (CatchupConn, error)
	DialDb(context.Context) (DbCoordConn, error)
	// DialExecs blocks and returns a set of executor the handler can spare for the next validation/nomination task.
	DialExecs(context.Context) ([]ExecCoordConn, error)

	RepairMissingTxs(
		ctx context.Context,
		from proto.ValidatorPublicInfo,
		missing proto.MissingTransactions,
	) ([]proto.SignedTransaction, error)

	ScheduleTxs(
		ctx context.Context,
		txs []*proto.SignedTransaction,
	) ([]*proto.SignedTransaction, []*proto.RejectedTransaction)

	RejectedTx(context.Context, *proto.RejectedTransaction)
	Committed(context.Context, *proto.CatchupBlock)

	Emit(ctx context.Context, event any)
}

type ID64 int64

func (id ID64) String() string { return fmt.Sprintf("%x", int64(id)) }

type SubmittedTxMempool interface {
	ListTxs(context.Context) []*proto.SignedTransaction
	PartialLookupTxs(ctx context.Context, hashes []proto.SignedTransactionHash) []*proto.SignedTransaction
	LookupTxs(context.Context, []proto.SignedTransactionHash) ([]*proto.SignedTransaction, bool)
	WaitTxs(context.Context, []proto.SignedTransactionHash) ([]*proto.SignedTransaction, bool)
	RemoveTx(context.Context, proto.SignedTransactionHash)
	TxCount() int
}

type CommittedTxMempool interface {
	AddCommittedTx(context.Context, proto.SignedTransactionHash)
	IsCommittedTx(context.Context, proto.SignedTransactionHash) bool
	CommittedTxCount() int
}

type CatchupConn interface {
	proto.Catchup1
	Close()
}

type DbCoordConn interface {
	proto.DbCoord1
	Close()
}

type ExecCoordConn interface {
	proto.ExecutorCoord1
	Close()
}

type Consensus interface {
	Nominate(ctx context.Context, slot scpapi.Slot, value xdr.XdrType, prev xdr.XdrType) bool
	StopNomination(ctx context.Context, slot scpapi.Slot)
}

type Disseminator interface {
	Disseminate(ctx context.Context, submitted *proto.SubmittedTransactions)
}

type Submitter interface {
	SubmitTxs(ctx context.Context, txs ...*proto.SignedTransaction)
}
