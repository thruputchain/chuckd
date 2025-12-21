package dogwood

import (
	"context"

	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libmem"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
)

type Multiplex struct {
	router   router.Router
	sender   *TransmissionSender
	receiver *TransmissionReceiver
	nav      *Navigator
	mMsg     libxdr.XdrMarshaler
	mTree    libxdr.XdrMarshaler
}

func NewMultiplex(
	ctx context.Context,
	router router.Router,
	s *TransmissionSender,
	r *TransmissionReceiver,
	n *Navigator,
) *Multiplex {
	return &Multiplex{
		router:   router,
		sender:   s,
		receiver: r,
		nav:      n,
		mMsg:     libxdr.NewXdrMarshaler(),
		mTree:    libxdr.NewXdrMarshaler(),
	}
}

func (x *Multiplex) Start(context.Context) {}

func (x *Multiplex) Receive(ctx context.Context) ([]byte, proto.ValidatorID, error) {
loop:
	for {
		typ, pay, from, err := x.receiver.Receive(ctx)
		if err != nil {
			return nil, proto.ValidatorID{}, err
		}

		switch typ {
		case proto.DogwoodTransmissionPayload:
			liblog.Tracef("dogwood multiplex at %v: receiving payload, message size %d", x.selfString(), len(pay))
			return pay, from, nil
		case proto.DogwoodTransmissionRoute:
			x.receiveRouting(ctx, pay)
			continue loop
		default: // unknown type
		}
	}
}

func (x *Multiplex) selfString() string {
	return protoutil.VIDString(protoutil.PublicVID(x.router.Self()))
}

func (x *Multiplex) DisseminateRoutes(ctx context.Context, ts []*dogwoodutil.LinkedTreeNode) {
	liblog.Debugf("dogwood multiplex at %v: disseminating %d routes", x.selfString(), len(ts))
	rs := make([]proto.DogwoodTreeNodes, len(ts))
	for i := range rs {
		rs[i] = ts[i].TreeNodes(x.mTree)
	}
	x.sendRouting(ctx, proto.DogwoodRouting{Routes: rs})
}

func (x *Multiplex) sendRouting(ctx context.Context, routing proto.DogwoodRouting) {
	p := x.mMsg.MarshalXdr(&routing)
	x.sender.Send(ctx, proto.DogwoodTransmissionRoute, p, true)
}

func (x *Multiplex) receiveRouting(ctx context.Context, p []byte) {
	liblog.Tracef("dogwood multiplex at %v: receiving routes, message size %d", x.selfString(), len(p))

	routing := proto.DogwoodRouting{}

	err := libxdr.UnmarshalXdr(&routing, p)
	if err != nil {
		liblog.Errorf("dogwood multiplex at %v: unmarshalling routing message: %v", x.selfString(), err)
		return
	}

	for _, treeNodes := range routing.Routes {
		linked, err := dogwoodutil.BuildLinkedTree(x.mTree, treeNodes)
		if err != nil {
			liblog.Errorf("dogwood multiplex at %v: encountered invalid routing tree: %v", x.selfString(), err)
			continue
		}
		h := dogwoodutil.TreeNodeHash(x.mTree, treeNodes[0])
		liblog.Tracef("dogwood multiplex at %v: caching route %v", x.selfString(), libxdr.HashString(h))
		x.nav.CacheRoute(ctx, linked)
	}
}

func (x *Multiplex) Send(ctx context.Context, pay libmem.BorrowedBytes) error {
	x.sender.Send(ctx, proto.DogwoodTransmissionPayload, pay, false)
	return nil
}
