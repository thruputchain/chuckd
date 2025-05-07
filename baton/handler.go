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

	RestoreDb(context.Context) (proto.BlockHeight, proto.BlockHash, error) // implemented by the coordinator service

	DialCatchup(context.Context, proto.ValidatorPublicInfo) (CatchupConn, error)
	DialDb(context.Context) (DbCoordConn, error)
	DialExec(context.Context)
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
