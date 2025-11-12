package hoglock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

const LeaseRenewInterval = 1 * time.Second
const LeaseTTL = 3 * time.Second

type Service struct {
	lk      sync.Mutex
	cond    *sync.Cond
	objects map[proto.LockObject]proto.LockLease
	leases  map[proto.LockLease]time.Time
}

func NewService() *Service {
	x := &Service{
		objects: map[proto.LockObject]proto.LockLease{},
		leases:  map[proto.LockLease]time.Time{},
	}
	x.cond = sync.NewCond(&x.lk)
	return x
}

func (x *Service) Start(ctx context.Context) {
	go func() {
		for {
			time.Sleep(LeaseTTL)
			x.expireLeases(ctx)
		}
	}()
}

func (x *Service) acquireLock(_ context.Context, h proto.LockLease, o proto.LockObject) {
	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()
	for {
		if _, locked := x.objects[o]; locked {
			x.cond.Wait()
		} else {
			x.objects[o] = h
		}
	}
}

func (x *Service) expireLeases(ctx context.Context) {
	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()
	now := time.Now()
	for h, lastRenewed := range x.leases {
		if now.Sub(lastRenewed) > LeaseTTL {
			x.releaseLease(ctx, h)
		}
	}
}

func (x *Service) releaseLease(_ context.Context, h proto.LockLease) {
	for o, oh := range x.objects {
		if h == oh {
			delete(x.objects, o)
		}
	}
	delete(x.leases, h)
}

// --- rpc ---

func (x *Service) Hoglock_null(context.Context) error {
	return nil
}

func (x *Service) Hoglock_lock(ctx context.Context, o proto.LockObjects) (proto.LockLease, error) {
	if len(o) == 0 {
		return "", nil
	}

	h := randomHandle()
	sort.Strings(o)
	for _, o := range o {
		x.acquireLock(ctx, h, o)
	}

	x.lk.Lock()
	defer x.lk.Unlock()
	x.leases[h] = time.Now()

	return h, nil
}

func (x *Service) Hoglock_renew(ctx context.Context, h proto.LockLease) error {
	x.lk.Lock()
	defer x.lk.Unlock()
	_, known := x.leases[h]
	if !known {
		return nil
	}
	x.leases[h] = time.Now()
	return nil
}

func (x *Service) Hoglock_check(ctx context.Context, h proto.LockLease) (bool, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	_, active := x.leases[h]
	return active, nil
}

func randomHandle() proto.LockLease {
	const n = 8
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
