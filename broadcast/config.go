package broadcast

import (
	proto "github.com/groundhogchain/protocol-go"
)

var DefaultBroadcastConfig = proto.BroadcastConfig{
	Multicast_tree_ttl_msec: 2e3,
	Chunk_size:              10e3,
}
