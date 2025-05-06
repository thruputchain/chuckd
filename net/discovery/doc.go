package discovery

/*

     # Discovery system

	At each node, the discovery system maintains a list of time-stamped and
	signed quorum slices, together with the identities of the signing
	validators. For short, quorum slices.

	Its API comprises a single call ‘update’ which expects a list of quorum
	slices, and returns the same. Nodes call ‘update’ to share the list of
	quorum slices they know, and obtain in return the slices known to the
	receiving node.

	Nodes maintain a list of known quorum slices. Upon `update`, they merge the
	given list of slices into the known slices by preferring newer timestamps.

	Every few minutes, nodes invoke the `update` rpc of all nodes in their own
	quorums in parallel, using a throttle, to share their most recent view on
	the quorum slices in the entire system.

	To its owner, upon request, the discovery system provides a current
	snapshot of known list of slices. This information, which describes an FBA
	graph, is used downstream by the broadcast system.

	The discovery protocol API is specified in
     [discovery.x](https://github.com/groundhogchain/protocol/blob/main/discovery.x).

*/
