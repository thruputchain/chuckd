package protoutil

import (
	"crypto/sha256"

	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

func MulticastTreeNodeHash(treeNode proto.MulticastTreeNode) proto.MulticastTreeHash {
	return sha256.Sum256(xdrutil.MarshalXdr(&treeNode))
}
