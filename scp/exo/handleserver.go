package exo

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/scp/scpapi"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

type SCPHandler struct {
	env  scpapi.SCPEmitEnvelope
	qset scpapi.SCPGetQuorumSet
	scp  scpapi.SCPHandler
}

func NewSCPHandler(
	ctx context.Context,
	env scpapi.SCPEmitEnvelope,
	qset scpapi.SCPGetQuorumSet,
	scp scpapi.SCPHandler,
) *SCPHandler {
	return &SCPHandler{env: env, qset: qset, scp: scp}
}

func (x *SCPHandler) Start(ctx context.Context, addr string) {
	// must use parallel because validations are long-running
	rpcutil.StartParallel(ctx, libnet.TCPNet{}, addr, proto.SCP_Handle1_Server{Srv: x})
}

func (x *SCPHandler) Scp_handle_null(context.Context) error {
	return nil
}

func (x *SCPHandler) Scp_handle_emit_envelope(ctx context.Context, env proto.SCPEnvelope) error {
	return x.env.EmitEnvelope(ctx, env)
}

func (x *SCPHandler) Scp_handle_validate_ballot_value(ctx context.Context, slotIndex proto.SCPSlotIndex, value proto.SCPValue) (proto.SCPValidationLevel, error) {
	return x.scp.ValidateBallotValue(ctx, slotIndex, value), nil
}

func (x *SCPHandler) Scp_handle_validate_nomination_value(ctx context.Context, slotIndex proto.SCPSlotIndex, value proto.SCPValue) (proto.SCPValidationLevel, error) {
	return x.scp.ValidateNominationValue(ctx, slotIndex, value), nil
}

func (x *SCPHandler) Scp_handle_extract_valid_value(ctx context.Context, slotIndex proto.SCPSlotIndex, value proto.SCPValue) (proto.SCPOptValue, error) {
	return x.scp.ExtractValidValue(ctx, slotIndex, value), nil
}

func (x *SCPHandler) Scp_handle_combine_candidates(ctx context.Context, slotIndex proto.SCPSlotIndex, values proto.SCPValues) (proto.SCPOptValue, error) {
	v := x.scp.CombineCandidates(ctx, slotIndex, values)
	return &v, nil
}

func (x *SCPHandler) Scp_handle_value_externalized(ctx context.Context, slotIndex proto.SCPSlotIndex, value proto.SCPValue) error {
	x.scp.ExternalizedValue(ctx, slotIndex, value)
	return nil
}

func (x *SCPHandler) Scp_handle_get_quorum_set(ctx context.Context, hash proto.SCPHash) (proto.SCPOptQuorumSet, error) {
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
