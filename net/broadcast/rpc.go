package broadcast

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

func (svc *BroadcastService) Broadcast_null(ctx context.Context) error {
	return nil
}

func (svc *BroadcastService) Broadcast_deliver(ctx context.Context, delivery Delivery) error {

	liblog.Debugf("receiving message %d on channel %s at %s from %s",
		delivery.Msg.Header.Seqno,
		protoutil.ChannelString(delivery.Msg.Header.Channel),
		protoutil.VPublicShortString(svc.ValidatorPublicInfo()),
		protoutil.VPublicShortString(delivery.Relayer),
	)

	if !protoutil.VerifyDelivery(delivery) {
		liblog.Errorf("discarding (verification) message %d on channel %s at %s from %s",
			delivery.Msg.Header.Seqno,
			protoutil.ChannelString(delivery.Msg.Header.Channel),
			protoutil.VPublicShortString(svc.ValidatorPublicInfo()),
			protoutil.VPublicShortString(delivery.Relayer),
		)
		return nil //discard
	}

	channel := delivery.Msg.Header.Channel

	svc.lk.Lock()

	if s := svc.sending[channel]; s != nil {
		svc.lk.Unlock()
		liblog.Errorf("discarding (sender) message %d on channel %s at %s from %s",
			delivery.Msg.Header.Seqno,
			protoutil.ChannelString(delivery.Msg.Header.Channel),
			protoutil.VPublicShortString(svc.ValidatorPublicInfo()),
			protoutil.VPublicShortString(delivery.Relayer),
		)
		return nil // discard

	} else if a := svc.accepting[channel]; a != nil {
		svc.lk.Unlock()
		a.deliver(ctx, delivery)

	} else if r := svc.receiving[channel]; r != nil {
		svc.lk.Unlock()
		r.deliver(ctx, delivery) // deliver to app and schedule relay

	} else {
		// ask the application whether the broadcast is acceptable
		if !svc.disc.IsAcceptable(channel, delivery.Msg.Header.Seqno, delivery.Msg.Payload) {
			svc.lk.Unlock()
			return nil // discard
		}
		svc.accepting[channel] = newAccepting(svc.ctx, svc, delivery)
		svc.lk.Unlock()
	}
	return nil
}

// XXX: DOS attacks?
func (svc *BroadcastService) Broadcast_reconcile(ctx context.Context, x ReconcileMessages) (ReconciledMessages, error) {

	svc.lk.Lock()
	defer svc.lk.Unlock()

	if s := svc.sending[x.Channel]; s != nil {
		return ReconciledMessages{Msgs: s.reconcile(x.Msg_hashes)}, nil
	} else if r := svc.receiving[x.Channel]; r != nil {
		return ReconciledMessages{Msgs: r.reconcile(x.Msg_hashes)}, nil
	} else {
		return ReconciledMessages{}, nil
	}
}

func (svc *BroadcastService) Broadcast_expand(ctx context.Context, x MulticastTreeHash) (MulticastTreeNodes, error) {
	return svc.findTree(x), nil
}

func (svc *BroadcastService) findTree(h MulticastTreeHash) MulticastTreeNodes {
	svc.lk.Lock()
	defer svc.lk.Unlock()
	nodes, _ := svc.trees.Find(h)
	return nodes
}
