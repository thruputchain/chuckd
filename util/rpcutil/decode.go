package rpcutil

import (
	"context"

	"github.com/thruputchain/lib-go/liblog"
	"github.com/xdrpp/goxdr/xdr"
)

// LogDecodeErrors logs malformed RPC arguments before the RPC decoder rejects
// them. Unknown public key union tags fail decoding before reaching a handler.
func LogDecodeErrors(ctx context.Context, srv xdr.XdrSrv) xdr.XdrSrv {
	return loggingServer{XdrSrv: srv, ctx: ctx}
}

type loggingServer struct {
	xdr.XdrSrv
	ctx context.Context
}

func (s loggingServer) GetProc(number uint32) xdr.XdrSrvProc {
	proc := s.XdrSrv.GetProc(number)
	if proc == nil {
		return nil
	}
	return loggingServerProc{XdrSrvProc: proc, ctx: s.ctx}
}

type loggingServerProc struct {
	xdr.XdrSrvProc
	ctx context.Context
}

func (p loggingServerProc) GetArg() xdr.XdrType {
	return loggingValue{XdrType: p.XdrSrvProc.GetArg(), ctx: p.ctx}
}

// LogReplyDecodeErrors also covers unknown key tags in RPC replies, which the
// RPC decoder rejects before returning a result to the caller.
func LogReplyDecodeErrors(send xdr.XdrSendCall) xdr.XdrSendCall {
	return loggingSender{XdrSendCall: send}
}

type loggingSender struct{ xdr.XdrSendCall }

func (s loggingSender) SendCall(ctx context.Context, proc xdr.XdrProc) error {
	return s.XdrSendCall.SendCall(ctx, loggingClientProc{XdrProc: proc, ctx: ctx})
}

type loggingClientProc struct {
	xdr.XdrProc
	ctx context.Context
}

func (p loggingClientProc) GetRes() xdr.XdrType {
	return loggingValue{XdrType: p.XdrProc.GetRes(), ctx: p.ctx}
}

type loggingValue struct {
	xdr.XdrType
	ctx context.Context
}

func (v loggingValue) XdrMarshal(codec xdr.XDR, name string) {
	defer func() {
		if failure := recover(); failure != nil {
			if err, ok := failure.(xdr.XdrError); ok {
				liblog.Infof(v.ctx, "rejecting malformed RPC %s: %v", v.XdrTypeName(), err)
			}
			// The RPC driver's safeMarshal converts XdrError into GARBAGE_ARGS
			// or GARBAGE_RET. Preserve that behavior and other panic handling.
			panic(failure)
		}
	}()
	v.XdrType.XdrMarshal(codec, name)
}
