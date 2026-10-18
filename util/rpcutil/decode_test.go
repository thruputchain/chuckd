package rpcutil

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/thruputchain/lib-go/libxdr"
	proto "github.com/thruputchain/thruput-protocol-go"
	"github.com/xdrpp/goxdr/rpc"
	"github.com/xdrpp/goxdr/xdr"
)

func decodeLogContext(t *testing.T) (context.Context, *bytes.Buffer) {
	t.Helper()
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(oldLevel) })
	logs := &bytes.Buffer{}
	return zerolog.New(logs).WithContext(context.Background()), logs
}

func TestRPCRejectsUnknownPublicKeyTagWithInfoLog(t *testing.T) {
	ctx, logs := decodeLogContext(t)
	service := LogDecodeErrors(ctx, proto.Catchup1_Server{})
	var server rpc.RpcSrv
	server.Register(service)
	call := &rpc.Rpc_msg{}
	call.Body.Mtype = rpc.CALL
	call.Body.Cbody().Rpcvers = 2
	call.Body.Cbody().Prog = service.Prog()
	call.Body.Cbody().Vers = service.Vers()
	call.Body.Cbody().Proc = 10 // catchup_solicit
	m := libxdr.NewXdrMarshaler()
	encoded := m.MarshalXdr(&proto.SignedCatchupBlocksRequest{})
	defer encoded.Recycle()
	bad := bytes.Clone(encoded.Bytes())
	// The first field is the signer's public key union tag.
	binary.BigEndian.PutUint32(bad, 123)
	reply, proc := server.GetProc(call, xdr.XdrIn{In: bytes.NewReader(bad)})
	if proc != nil || reply.Body.Rbody().Areply().Reply_data.Stat != rpc.GARBAGE_ARGS {
		t.Fatal("unknown public key tag was not rejected as malformed RPC arguments")
	}
	if !strings.Contains(logs.String(), `"level":"info"`) || !strings.Contains(logs.String(), "rejecting malformed RPC") {
		t.Fatalf("expected Info rejection log, got %s", logs.String())
	}
	logs.Reset()
	reply, proc = server.GetProc(call, xdr.XdrIn{In: bytes.NewReader(encoded.Bytes())})
	if proc == nil || !rpc.IsSuccess(reply) {
		t.Fatal("valid request was rejected after malformed request")
	}
	if logs.Len() != 0 {
		t.Fatalf("unexpected rejection log for valid request: %s", logs.String())
	}
}

type decodingSender struct{ payload []byte }

func (s decodingSender) SendCall(_ context.Context, proc xdr.XdrProc) error {
	return libxdr.UnmarshalXdrEOF(proc.GetRes(), s.payload)
}

func TestRPCReplyRejectsUnknownPublicKeyTagWithInfoLog(t *testing.T) {
	ctx, logs := decodeLogContext(t)
	m := libxdr.NewXdrMarshaler()
	encoded := m.MarshalXdr(&proto.SignedCatchupBlocksResult{})
	defer encoded.Recycle()
	bad := bytes.Clone(encoded.Bytes())
	binary.BigEndian.PutUint32(bad, 123)
	client := proto.Catchup1_Client{Send: LogReplyDecodeErrors(decodingSender{bad})}
	if _, err := client.Catchup_solicit(ctx, proto.SignedCatchupBlocksRequest{}); err == nil {
		t.Fatal("unknown reply public key tag was accepted")
	}
	if !strings.Contains(logs.String(), `"level":"info"`) || !strings.Contains(logs.String(), "rejecting malformed RPC") {
		t.Fatalf("expected Info rejection log, got %s", logs.String())
	}
	logs.Reset()
	client.Send = LogReplyDecodeErrors(decodingSender{encoded.Bytes()})
	if _, err := client.Catchup_solicit(ctx, proto.SignedCatchupBlocksRequest{}); err != nil {
		t.Fatal(err)
	}
	if logs.Len() != 0 {
		t.Fatalf("unexpected rejection log for valid reply: %s", logs.String())
	}
}
