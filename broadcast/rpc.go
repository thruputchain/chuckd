package broadcast

import "github.com/groundhogchain/chuckd/util/protoutil"

func (svc *BroadcastService) Broadcast_null() {}

func (svc *BroadcastService) Broadcast_deliver(delivery Delivery) {

	if !protoutil.VerifyDelivery(delivery) {
		return //discard
	}

	channel := delivery.Msg.Header.Channel

	svc.lk.Lock()
	defer svc.lk.Unlock()

	if s := svc.sending[channel]; s != nil {
		return // discard

	} else if a := svc.accepting[channel]; a != nil {
		a.deliver(delivery)

	} else if r := svc.receiving[channel]; r != nil {
		r.deliver(delivery) // deliver to app and schedule relay

	} else {
		// ask the application whether the broadcast is acceptable
		if !svc.disc.IsAcceptable(channel, delivery.Msg.Header.Seqno, delivery.Msg.Payload) {
			return // discard
		}
		svc.accepting[channel] = newAccepting(svc, delivery)

	}
}

// XXX: DOS attacks?
func (svc *BroadcastService) Broadcast_reconcile(x ReconcileMessages) ReconciledMessages {

	svc.lk.Lock()
	defer svc.lk.Unlock()

	if s := svc.sending[x.Channel]; s != nil {
		return ReconciledMessages{Msgs: s.reconcile(x.Msg_hashes)}
	} else if r := svc.receiving[x.Channel]; r != nil {
		return ReconciledMessages{Msgs: r.reconcile(x.Msg_hashes)}
	} else {
		return ReconciledMessages{}
	}
}

func (svc *BroadcastService) Broadcast_expand(x MulticastTreeHash) MulticastTreeNodes {
	return svc.findTree(x)
}
