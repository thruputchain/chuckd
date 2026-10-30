package systemfixed

// Routing tree constants.
const (
	// MaxRouteNodes is the maximum number of nodes in a routing tree.
	MaxRouteNodes = 1_000
)

// Parallel batch segmentation constants.
const (
	NominationTransactionHashBatchMaxSegments      = 20
	NominationNondeterministicDataBatchMaxSegments = 20

	UnvettedTxBatchMaxSegments  = 20
	MultiplexTxBatchMaxSegments = 20

	CatchupTxBatchMaxSegments = 20
	RepairTxBatchMaxSegments  = 20
)
