package systemfixed

const (
	// MaxQuorumSetNestingDepth is the maximum nesting depth for quorum sets,
	// HOWEVER this bound is currently enforced AFTER deserialization, so
	// deserialization (which is recursive) is NOT safe from stack overflow.
	// XXX(SECURITY): transition to non-recursive quorum set structures across validators,
	// then convert them to recursive ones after entering the validator system.
	MaxQuorumSetNestingDepth = 3
)
