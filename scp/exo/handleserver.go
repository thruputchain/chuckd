package exo

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type SCPHandler struct {
}

func (x *SCPHandler) Scp_handle_null(context.Context) error {
	return nil
}

func (x *SCPHandler) Scp_handle_emit_envelope(context.Context, proto.SCPEnvelope) error {
	panic("XXX")
}

func (x *SCPHandler) Scp_handle_validate_ballot_value(context.Context, proto.SCPSlotIndex, proto.SCPValue) (proto.SCPValidationLevel, error) {
	panic("XXX")
}

func (x *SCPHandler) Scp_handle_validate_nomination_value(context.Context, proto.SCPSlotIndex, proto.SCPValue) (proto.SCPValidationLevel, error) {
	panic("XXX")
}

func (x *SCPHandler) Scp_handle_extract_valid_value(context.Context, proto.SCPSlotIndex, proto.SCPValue) (proto.SCPOptValue, error) {
	panic("XXX")
}

func (x *SCPHandler) Scp_handle_combine_candidates(context.Context, proto.SCPSlotIndex, proto.SCPValues) (proto.SCPOptValue, error) {
	panic("XXX")
}

func (x *SCPHandler) Scp_handle_value_externalized(context.Context, proto.SCPSlotIndex, proto.SCPValue) error {
	panic("XXX")
}

func (x *SCPHandler) Scp_handle_get_quorum_set(context.Context, proto.SCPHash) (proto.SCPOptQuorumSet, error) {
	panic("XXX")
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
