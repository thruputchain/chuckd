package scpd

import (
	"context"

	"github.com/groundhogchain/chuckd/sub/scp/scpkit"
	"github.com/groundhogchain/chuckd/util/netutil"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	"github.com/groundhogchain/lib-go/libnet"
	"github.com/groundhogchain/lib-go/librpc"
	proto "github.com/groundhogchain/protocol-go"
)

type SCPHandler struct {
	ready chan struct{}
	env   scpkit.SCPEmitEnvelope // implemented by multiplex
	qset  scpkit.SCPGetQuorumSet // implemented by discovery
	scp   scpkit.SCPHandler      // implemented by baton
}

func NewSCPHandler(ctx context.Context) *SCPHandler {
	return &SCPHandler{ready: make(chan struct{})}
}

func (x *SCPHandler) SetHandlers(
	env scpkit.SCPEmitEnvelope,
	qset scpkit.SCPGetQuorumSet,
	scp scpkit.SCPHandler,
) {
	x.env = env
	x.qset = qset
	x.scp = scp
	close(x.ready)
}

func (x *SCPHandler) Start(ctx context.Context, addr string) {
	// must use parallel because validations are long-running
	librpc.StartParallel(ctx, libnet.TCPNet{}, netutil.MaxMessageSize, rpcutil.BaseDriverRegime, addr, proto.SCP_Handle1_Server{Srv: x})
}

func (x *SCPHandler) Scp_handle_null(context.Context) error {
	return nil
}

func (x *SCPHandler) Scp_handle_emit_envelope(ctx context.Context, env proto.SCPEnvelope) error {
	<-x.ready
	return x.env.EmitEnvelope(ctx, env)
}

func (x *SCPHandler) Scp_handle_validate_ballot_value(ctx context.Context, slotIndex proto.SCPSlotIndex, value proto.SCPValue) (proto.SCPValidationLevel, error) {
	<-x.ready
	return x.scp.ValidateBallotValue(ctx, slotIndex, value), nil
}

func (x *SCPHandler) Scp_handle_validate_nomination_value(ctx context.Context, slotIndex proto.SCPSlotIndex, value proto.SCPValue) (proto.SCPValidationLevel, error) {
	<-x.ready
	return x.scp.ValidateNominationValue(ctx, slotIndex, value), nil
}

func (x *SCPHandler) Scp_handle_extract_valid_value(ctx context.Context, slotIndex proto.SCPSlotIndex, value proto.SCPValue) (proto.SCPOptValue, error) {
	<-x.ready
	return x.scp.ExtractValidValue(ctx, slotIndex, value), nil
}

func (x *SCPHandler) Scp_handle_combine_candidates(ctx context.Context, slotIndex proto.SCPSlotIndex, values proto.SCPValues) (proto.SCPOptValue, error) {
	<-x.ready
	v := x.scp.CombineCandidates(ctx, slotIndex, values)
	return &v, nil
}

func (x *SCPHandler) Scp_handle_value_externalized(ctx context.Context, slotIndex proto.SCPSlotIndex, value proto.SCPValue) error {
	<-x.ready
	x.scp.ExternalizedValue(ctx, slotIndex, value)
	return nil
}

func (x *SCPHandler) Scp_handle_get_quorum_set(ctx context.Context, hash proto.SCPHash) (proto.SCPOptQuorumSet, error) {
	<-x.ready
	return x.qset.GetQuorumSet(ctx, hash)
}

func (x *SCPHandler) Scp_handle_nominating_value(context.Context, proto.SCPSlotIndex, proto.SCPValue) error {
	return nil
}

func (x *SCPHandler) Scp_handle_updated_candidate_value(context.Context, proto.SCPSlotIndex, proto.SCPValue) error {
	return nil
}

func (x *SCPHandler) Scp_handle_started_ballot_proto(context.Context, proto.SCPSlotIndex, proto.SCPBallot) error {
	return nil
}

func (x *SCPHandler) Scp_handle_accepted_ballot_prepared(context.Context, proto.SCPSlotIndex, proto.SCPBallot) error {
	return nil
}

func (x *SCPHandler) Scp_handle_confirmed_ballot_prepared(context.Context, proto.SCPSlotIndex, proto.SCPBallot) error {
	return nil
}

func (x *SCPHandler) Scp_handle_accepted_commit(context.Context, proto.SCPSlotIndex, proto.SCPBallot) error {
	return nil
}

func (x *SCPHandler) Scp_handle_ballot_did_hear_from_quorum(context.Context, proto.SCPSlotIndex, proto.SCPBallot) error {
	return nil
}
