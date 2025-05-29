package mempool

import (
	"context"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/libtime"
	proto "github.com/groundhogchain/protocol-go"
)

type CommittedMempool struct {
	time      libtime.Time
	ttl       time.Duration
	lk        sync.Mutex
	committed map[proto.SHA256Hash]time.Time
}

func NewCommittedMempoolReal(ctx context.Context, ttl time.Duration) *CommittedMempool {
	return NewCommittedMempool(ctx, libtime.RealTime{}, ttl)
}

func NewCommittedMempool(ctx context.Context, time_ libtime.Time, ttl time.Duration) *CommittedMempool {
	mp := &CommittedMempool{
		time:      time_,
		ttl:       ttl,
		committed: map[proto.SHA256Hash]time.Time{},
	}
	go mp.supervise(ctx)
	return mp
}

func (mp *CommittedMempool) CommittedTxCount() int {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	return len(mp.committed)
}

func (mp *CommittedMempool) AddCommittedTx(ctx context.Context, k proto.SignedTransactionHash) {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	mp.committed[k] = mp.time.Now()
}

func (mp *CommittedMempool) IsCommittedTx(ctx context.Context, k proto.SignedTransactionHash) bool {
	mp.lk.Lock()
	defer mp.lk.Unlock()
	_, ok := mp.committed[k]
	return ok
}

func (mp *CommittedMempool) supervise(_ context.Context) {
	for {
		mp.time.Sleep(mp.ttl)
		now := mp.time.Now()
		mp.lk.Lock()
		for k, t := range mp.committed {
			if now.Sub(t) > mp.ttl {
				delete(mp.committed, k)
			}
		}
		mp.lk.Unlock()
	}
}
