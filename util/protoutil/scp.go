package protoutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

func SCPValueDisplay(value *proto.ConsensusValue) string {
	h := sha256.New()
	libxdr.MarshalXdrWriter(h, value)
	vhash := h.Sum(nil)
	return fmt.Sprintf("%x", vhash[:3])
}

func scpHexAbbrev(bytes []byte) string {
	n := len(bytes)
	if n > 3 {
		n = 3
	}
	return hex.EncodeToString(bytes[:n])
}

func scpValueToString(value proto.SCPValue) string {
	hash := sha256.Sum256(value)
	return fmt.Sprintf("hov:%s", scpHexAbbrev(hash[:]))
}

func scpBallotToString(ballot proto.SCPBallot) string {
	return fmt.Sprintf("<%d,%s>", ballot.Counter, scpValueToString(ballot.Value))
}

func scpOptBallotToString(ballot *proto.SCPBallot) string {
	if ballot == nil {
		return "<,>"
	}
	return scpBallotToString(*ballot)
}

func scpNodeIDToString(nodeID proto.SCPNodeID) string {
	if nodeID.Scheme != proto.Ed25519 {
		return fmt.Sprintf("unsupported public key scheme:%d", nodeID.Scheme)
	}
	validatorID := NodeToValidatorID(nodeID)
	return scpHexAbbrev(validatorID[:])
}

func SCPEnvelopeDisplay(env proto.SCPEnvelope) string {
	statement := env.Statement
	var result strings.Builder
	fmt.Fprintf(&result, "{ENV@%s |  i: %d", scpNodeIDToString(statement.Node_id), statement.Slot_index)

	switch statement.Pledges.Stmt_type {
	case proto.SCP_ST_PREPARE:
		prepare := statement.Pledges.Prepare()
		fmt.Fprintf(
			&result,
			" | PREPARE | D: %s | b: %s | p: %s | p': %s | c.n: %d | h.n: %d",
			scpHexAbbrev(prepare.Quorum_set_hash[:]),
			scpBallotToString(prepare.Ballot),
			scpOptBallotToString(prepare.Prepared),
			scpOptBallotToString(prepare.Prepared_prime),
			prepare.N_c,
			prepare.N_h,
		)
	case proto.SCP_ST_CONFIRM:
		confirm := statement.Pledges.Confirm()
		fmt.Fprintf(
			&result,
			" | CONFIRM | D: %s | b: %s | p.n: %d | c.n: %d | h.n: %d",
			scpHexAbbrev(confirm.Quorum_set_hash[:]),
			scpBallotToString(confirm.Ballot),
			confirm.N_prepared,
			confirm.N_commit,
			confirm.N_h,
		)
	case proto.SCP_ST_EXTERNALIZE:
		externalize := statement.Pledges.Externalize()
		fmt.Fprintf(
			&result,
			" | EXTERNALIZE | c: %s | h.n: %d | (lastD): %s",
			scpBallotToString(externalize.Commit),
			externalize.N_h,
			scpHexAbbrev(externalize.Commit_quorum_set_hash[:]),
		)
	case proto.SCP_ST_NOMINATE:
		nominate := statement.Pledges.Nominate()
		fmt.Fprintf(&result, " | NOMINATE | D: %s | X: {", scpHexAbbrev(nominate.Quorum_set_hash[:]))
		for i, vote := range nominate.Votes {
			if i > 0 {
				result.WriteString(" ,")
			}
			fmt.Fprintf(&result, "'%s'", scpValueToString(vote))
		}
		result.WriteString("} | Y: {")
		for i, accepted := range nominate.Accepted {
			if i > 0 {
				result.WriteString(" ,")
			}
			fmt.Fprintf(&result, "'%s'", scpValueToString(accepted))
		}
		result.WriteString("}")
	default:
		return "SCPEnvelope{UNRECOGNIZED}"
	}

	result.WriteString(" }")
	return result.String()
}
