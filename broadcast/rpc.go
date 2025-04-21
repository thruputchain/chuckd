package broadcast

import (
	"context"

	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/util/protoutil"
)

func (svc *BroadcastService) Broadcast_null(ctx context.Context) error {
	return nil
}

func (svc *BroadcastService) Broadcast_deliver(ctx context.Context, delivery Delivery) error {

	log.Debugf("receiving message %d on channel %s at %s from %s",
		delivery.Msg.Header.Seqno,
		protoutil.ChannelString(delivery.Msg.Header.Channel),
		protoutil.ValidatorPublicShortString(svc.ValidatorPublicInfo()),
		protoutil.ValidatorPublicShortString(delivery.Relayer),
	)

	if !protoutil.VerifyDelivery(delivery) {
		return nil //discard
	}

	channel := delivery.Msg.Header.Channel

	svc.lk.Lock()
	defer svc.lk.Unlock()

	if s := svc.sending[channel]; s != nil {
		return nil // discard

	} else if a := svc.accepting[channel]; a != nil {
		a.deliver(delivery)

	} else if r := svc.receiving[channel]; r != nil {
		r.deliver(ctx, delivery) // deliver to app and schedule relay

	} else {
		// ask the application whether the broadcast is acceptable
		if !svc.disc.IsAcceptable(channel, delivery.Msg.Header.Seqno, delivery.Msg.Payload) {
			return nil // discard
		}
		svc.accepting[channel] = newAccepting(svc, delivery)

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
