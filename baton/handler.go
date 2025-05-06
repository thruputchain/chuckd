package baton

import (
	"context"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

type Handler interface {
	Time
	RestoreDb(context.Context) (proto.BlockHeight, proto.BlockHash, error) // implemented by the coordinator service
	DialCatchup(context.Context, proto.ValidatorPublicInfo) (CatchupConn, error)
	DialDb(context.Context) (DbCoordConn, error)
	DialExec(context.Context)
	Mempool
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
