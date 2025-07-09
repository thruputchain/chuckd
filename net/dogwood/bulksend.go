package dogwood

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

type BulkSender interface {
	BulkSend(context.Context, []proto.DogwoodPackage) (proto.DogwoodStream, error)
}
