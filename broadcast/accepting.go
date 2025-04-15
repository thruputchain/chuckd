package broadcast

import (
	"context"
	"sync"

	"github.com/groundhogchain/chuckd/util/protoutil"
)

type accepting struct {
	svc     *BroadcastService
	channel Channel
	//
	lk         sync.Mutex
	deliveries []Delivery
}

func newAccepting(svc *BroadcastService, first Delivery) *accepting {
	a := &accepting{
		svc:     svc,
		channel: first.Msg.Header.Channel,
		//
		deliveries: []Delivery{first},
	}
	go a.accept(context.TODO())
	return a
}

// called inside a service lock!
func (a *accepting) deliver(delivery Delivery) {
	a.lk.Lock()
	defer a.lk.Unlock()
	a.deliveries = append(a.deliveries, delivery)
}

func (a *accepting) accept(ctx context.Context) {

	// if the multicast tree is not known, try to retrieve it
	a.lk.Lock()
	first := a.deliveries[0]
	a.lk.Unlock()

	treeNodes := first.Tree_nodes
	if treeNodes == nil {
		expanded := expandTree(ctx, first.Relayer.Address, a.channel.Tree_hash)
		treeNodes = &expanded
	}

	linked := protoutil.VerifyBuildLinkedTree(a.channel.Tree_hash, *treeNodes)
	children, ok := linked.FindChildren(a.svc.ValidatorPublicInfo())

	// if we fail to retrieve the tree and find out children in it, abort
	if !ok {
		a.svc.lk.Lock()
		defer a.svc.lk.Unlock()
		delete(a.svc.accepting, a.channel)
		return
	}

	// otherwise create a receiving structure

	// lock service and accepting structure
	a.svc.lk.Lock()
	a.lk.Lock()

	r := newReceiving(a.svc, a.channel, *treeNodes, children, a.deliveries)
	delete(a.svc.accepting, a.channel)
	a.svc.receiving[a.channel] = r
	a.svc.trees.Ref(a.channel.Tree_hash, *treeNodes)

	// unlock accepting structure and service
	a.lk.Unlock()
	a.svc.lk.Unlock()

	// notify app of accepted broadcast channel
	a.svc.acceptCh <- r
}

func expandTree(ctx context.Context, address string, treeHash MulticastTreeHash) MulticastTreeNodes {
	c := NewBroadcastClient(ctx, address)
	defer c.Close()
	return c.Broadcast_expand(treeHash) //XXX: what happens if a call fails or timeouts?
}
