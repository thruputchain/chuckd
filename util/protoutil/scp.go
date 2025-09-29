package protoutil

import (
	"fmt"

	proto "github.com/groundhogchain/protocol-go"
)

func SCPEnvelopeShortString(env proto.SCPEnvelope) string {
	switch env.Statement.Pledges.Stmt_type {
	case proto.SCP_ST_PREPARE:
		return fmt.Sprintf("SCPEnvelope{slot:%d, prepare}", env.Statement.Slot_index)
	case proto.SCP_ST_CONFIRM:
		return fmt.Sprintf("SCPEnvelope{slot:%d, confirm}", env.Statement.Slot_index)
	case proto.SCP_ST_EXTERNALIZE:
		return fmt.Sprintf("SCPEnvelope{slot:%d, externalize}", env.Statement.Slot_index)
	case proto.SCP_ST_NOMINATE:
		return fmt.Sprintf("SCPEnvelope{slot:%d, nominate}", env.Statement.Slot_index)
	}
	return "SCPEnvelope{UNRECOGNIZED}"
}
