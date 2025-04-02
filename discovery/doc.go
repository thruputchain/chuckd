package discovery

// TODO:
// - only the tree root computes the multicast tree and sends its hash along with messages
//	- provide network call to retrieve multicast tree, if not known by some receiving node

/*

	# Discovery protocol specification

	## On the wire
	- share all quorums we know every few minutes by calling `update_quorums`
	- timestamps and quorums signed
	- respond with known quorums on `update_quorums`

	## Rationale

*/
