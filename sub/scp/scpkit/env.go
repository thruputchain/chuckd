package scpkit

import (
	proto "github.com/thruputchain/thruput-protocol-go"
)

func IsSaneSCPEnvelope(env *proto.SCPEnvelope) bool {
	switch env.Statement.Pledges.Stmt_type {
	case proto.SCP_ST_PREPARE:
		return true
	case proto.SCP_ST_CONFIRM:
		return true
	case proto.SCP_ST_EXTERNALIZE:
		return true
	case proto.SCP_ST_NOMINATE:
		return true
	}
	return false
}
