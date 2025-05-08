package baton

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

type Handler interface {
	Time
	Mempool

	Signer() proto.ValidatorPublicInfo
	SignXdr(xdr.XdrType) proto.Signature
	VerifyXdr(signer proto.ValidatorPublicKey, value xdr.XdrType, sig proto.Signature) bool

	ProvideCatchupSources(context.Context, int) []proto.ValidatorPublicInfo

	RestoreDb(context.Context) (proto.BlockHeight, proto.BlockHash, proto.BlockID, error) // implemented by the coordinator service
	// XXX: 64-bit block ids are too small for preventing collisions with large block counts
	GenerateBlockID(context.Context) proto.BlockID

	DialCatchup(context.Context, proto.ValidatorPublicInfo) (CatchupConn, error)

	DialDb(context.Context) (DbCoordConn, error)

	// DialExecs blocks and returns a set of executor the handler can spare for the next validation/proposal task.
	DialExecs(context.Context) ([]ExecCoordConn, error)
}

type Time interface {
	Now() time.Time
}

type Mempool interface {
	LookupBatch(context.Context, proto.BatchHash) ([]proto.ProposedTransaction, bool)
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
