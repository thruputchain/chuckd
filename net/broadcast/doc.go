// Package broadcast provides a system for broadcasting and reconciliation of opaque messages.
package broadcast

/*

     # Broadcast system

	The broadcast system provides a lossy, ordered broadcast channel
	abstraction. Channels are assigned random IDs and senders control their
	lifetime. During a channel’s lifetime, all nodes receive payload messages,
	and can request to reconcile missing subsequences of messages.

	Channels are created at the sending node by picking a random channel ID.
	Using the FBA-graph view, together with randomness seeded by the channel
	ID, the sender generates a random spanning tree of the nodes in its
	FBA-graph view, meeting some optimization criteria. The tree is encoded as
	a merkle tree of validator identities, and called a multicast tree.

	Broadcasting payload messages involves relaying them down the multicast
	tree (via the `deliver` rpc call) along with a header.
	- For the purpose of enabling reconciliation of missing messages, the
	  header contains sender ID, channel ID, hash of multicast tree, and (for
	  efficiency optimization) the first header may contain the multicast tree
	  itself.
	- For the purpose of integrity verification, the header contains the hash
	  of the previous message and header.

	When a node receives a message, it passes the channel ID, sequence number
	and payload along to the application layer. If the node doesn’t already
	know the multicast tree (in case it never received the first message on the
	channel), it reconstructs it by inquiring the message sender for it, who
	always has a cached copy (by induction). The node then relays the message
	along to its children in the multicast tree, and caches the multicast tree.

	Due to network or node runtime conditions, a node may not receive all (or
     any) messages sent on a channel. The application layer may determine that
     it needs to recover (i.e. reconcile) a missing set of channel messages,
     identified by the channel ID, the message sequence numbers, and the hashes
     of the missing header/message pairs. (For optimization, only the hashes of
     the last message in a missing contiguous sequence of messages is
     necessary.)

	The application layer determines the required information about missing
     messages, using an application-level method. In Groundhog, this information
     is derived from the contents of nominated blocks (txsets) which describe
     sets of transactions using the same semantic: a list of channel ID, bitmap
     of message sequence numbers, and hashes associated with each message. The
     messages known to the current node are removed from the txset to derive a
     description of the missing messages, using the same semantic.

	To reconcile missing messages:
	- If the node has a cached copy of the applicable channel multicast tree,
       it sends a reconciliation request to its parent (via `reconcile` rpc
       call), describing the missing set using channel ID, bitmap vector, and
       message hashes. The call blocks until the caller can provide the
       requested messages. The caller reconstructs the requested messages by
       utilizing a combination of its own message cache, and recursively
       invoking the reconciliation procedure.
	- If the node is unaware of the applicable multicast tree (implying it has
       never received any messages from this channel), it sends the
       reconciliation request to a randomly chosen node in the network. If that
       fails, as a final resort, it can request reconciliation from the channel
       root node itself.

	The discovery protocol API is specified in
     [broadcast.x](https://github.com/groundhogchain/protocol/blob/main/broadcast.x)

*/
