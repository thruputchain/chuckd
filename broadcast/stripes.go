package broadcast

import "sync"

type StripedBroadcast struct {
	b       Broadcast
	stripes []SendChannel
	lk      sync.Mutex
	next    int // next stripe
}

func MakeStripedSender(b Broadcast, nstripes int) *StripedBroadcast {
	x := &StripedBroadcast{
		b:       b,
		stripes: make([]SendChannel, nstripes),
	}
	for i := 0; i < nstripes; i++ {
		x.stripes[i] = b.MakeChannel()
	}
	return x
}

func (x *StripedBroadcast) Send(payload []byte) {
	x.lk.Lock()
	n := x.next
	x.next += 1
	x.lk.Unlock()
	x.stripes[n%len(x.stripes)].Send(payload)
}

func (x *StripedBroadcast) Close() {
	for _, s := range x.stripes {
		s.Close()
	}
}
