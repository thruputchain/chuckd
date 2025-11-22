package hoglock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/groundhogchain/chuckd/util/rpcutil"
	proto "github.com/groundhogchain/protocol-go"
)

const LeaseRenewInterval = 1 * time.Second
const LeaseTTL = 3 * time.Second

type Service struct {
	lk      sync.Mutex
	cond    *sync.Cond
	objects map[proto.LockObject]proto.LockLease
	leases  map[proto.LockLease]*lease
	//
	slk       sync.Mutex
	scond     *sync.Cond
	signalled map[proto.HogNamespace]struct{}
}

type lease struct {
	renewed time.Time
	objects proto.LockObjects
}

func NewService() *Service {
	x := &Service{
		objects: map[proto.LockObject]proto.LockLease{},
		leases:  map[proto.LockLease]*lease{},
		//
		signalled: map[proto.HogNamespace]struct{}{},
	}
	x.cond = sync.NewCond(&x.lk)
	x.scond = sync.NewCond(&x.slk)
	return x
}

func (x *Service) Start(ctx context.Context, n libnet.Network, address string) {
	go x.start(ctx)
	l := rpcutil.StartParallel(ctx, n, address, proto.HogLock1_Server{Srv: x})
	liblog.Infof("hog_lockd is listening on %v", l.Addr().String())
}

func (x *Service) start(ctx context.Context) {
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
			return
		}
	}
}

func (x *Service) expireLeases(ctx context.Context) {
	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()
	now := time.Now()
	for h, lease := range x.leases {
		if now.Sub(lease.renewed) > LeaseTTL {
			x.releaseLease(ctx, h)
		}
	}
}

func (x *Service) releaseLease(_ context.Context, h proto.LockLease) {
	liblog.Infof("expiring lease %v", h)
	lease := x.leases[h]
	delete(x.leases, h)

	slices.Reverse(lease.objects)
	liblog.Infof("releasing locks %v, while expiring lease %v", lease.objects, h)

	for _, o := range lease.objects {
		delete(x.objects, o)
	}
}

// --- rpc ---

func (x *Service) Hoglock_null(context.Context) error {
	return nil
}

func (x *Service) Hoglock_lock(ctx context.Context, o proto.LockObjects) (h proto.LockLease, err error) {

	liblog.Infof("hoglock_lock %v", o)
	defer func() {
		if err != nil {
			liblog.Errorf("hoglock_lock for %v not acquired (%v)", o, err)
		} else {
			liblog.Infof("hoglock_lock for %v acquired under lease %v", o, h)
		}
	}()

	if len(o) == 0 {
		return "", nil
	}

	h = randomHandle()
	sort.Strings(o)
	for _, o := range o {
		x.acquireLock(ctx, h, o)
	}

	x.lk.Lock()
	defer x.lk.Unlock()

	x.leases[h] = &lease{renewed: time.Now(), objects: o}

	return h, nil
}

func (x *Service) Hoglock_renew(ctx context.Context, h proto.LockLease) error {
	liblog.Infof("hoglock_renew %v", h)

	x.lk.Lock()
	defer x.lk.Unlock()
	lease, known := x.leases[h]
	if !known {
		return nil
	}
	lease.renewed = time.Now()
	return nil
}

func (x *Service) Hoglock_check(ctx context.Context, h proto.LockLease) (bool, error) {
	liblog.Infof("hoglock_check %v", h)

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

func (x *Service) Hoglock_signal(ctx context.Context, ns proto.HogNamespaces) error {

	x.slk.Lock()
	defer x.slk.Unlock()
	defer x.scond.Broadcast()

	for _, ns := range ns {
		x.signalled[ns] = struct{}{}
	}

	return nil
}

func (x *Service) Hoglock_wait(ctx context.Context, prefix proto.HogNamespaces) error {

	x.slk.Lock()
	defer x.slk.Unlock()

	for {
		if x.isSignalled(prefix) {
			return nil
		} else {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			x.cond.Wait()
		}
	}
}

func (x *Service) isSignalled(prefix proto.HogNamespaces) bool {
	uncovered := map[proto.HogNamespace]struct{}{}
	for _, prefix := range prefix {
		uncovered[prefix] = struct{}{}
	}
	for ns := range x.signalled {
		for u := range uncovered {
			if strings.HasPrefix(ns, u) {
				delete(uncovered, u)
			}
		}
	}
	return len(uncovered) == 0
}
