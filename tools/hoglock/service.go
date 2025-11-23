package hoglock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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
	lk        sync.Mutex
	cond      *sync.Cond
	hosts     map[proto.HogHost]proto.HogXID
	leases    map[proto.HogXID]*lease
	signalled map[proto.HogXID]map[proto.HogNamespace]struct{}
}

type lease struct {
	renewed time.Time
	hosts   proto.HogHosts
}

func NewService() *Service {
	x := &Service{
		hosts:  map[proto.HogHost]proto.HogXID{},
		leases: map[proto.HogXID]*lease{},
		//
		signalled: map[proto.HogXID]map[proto.HogNamespace]struct{}{},
	}
	x.cond = sync.NewCond(&x.lk)
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

func (x *Service) acquireLock(_ context.Context, xid proto.HogXID, host proto.HogHost) {
	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()
	for {
		if _, locked := x.hosts[host]; locked {
			x.cond.Wait()
		} else {
			x.hosts[host] = xid
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

func (x *Service) releaseLease(_ context.Context, xid proto.HogXID) {
	liblog.Infof("expiring lease %v", xid)

	delete(x.signalled, xid)

	lease := x.leases[xid]
	delete(x.leases, xid)

	slices.Reverse(lease.hosts)
	liblog.Infof("releasing locks %v, while expiring lease %v", lease.hosts, xid)

	for _, o := range lease.hosts {
		delete(x.hosts, o)
	}
}

// --- rpc ---

func (x *Service) Hoglock_null(context.Context) error {
	return nil
}

func (x *Service) Hoglock_lock(ctx context.Context, xid proto.HogXID, hosts proto.HogHosts) (err error) {

	liblog.Infof("hoglock_lock %v", hosts)
	defer func() {
		if err != nil {
			liblog.Errorf("hoglock_lock for %v not acquired (%v)", hosts, err)
		} else {
			liblog.Infof("hoglock_lock for %v acquired under lease %v", hosts, xid)
		}
	}()

	if len(hosts) == 0 {
		return nil
	}

	sort.Strings(hosts)
	for _, o := range hosts {
		x.acquireLock(ctx, xid, o)
	}

	x.lk.Lock()
	defer x.lk.Unlock()

	x.leases[xid] = &lease{renewed: time.Now(), hosts: hosts}

	return nil
}

func (x *Service) Hoglock_renew(ctx context.Context, xid proto.HogXID) error {
	liblog.Infof("hoglock_renew %v", xid)

	x.lk.Lock()
	defer x.lk.Unlock()
	lease, known := x.leases[xid]
	if !known {
		return nil
	}
	lease.renewed = time.Now()
	return nil
}

func (x *Service) Hoglock_check(ctx context.Context, xid proto.HogXID) (bool, error) {
	liblog.Infof("hoglock_check %v", xid)

	x.lk.Lock()
	defer x.lk.Unlock()
	_, active := x.leases[xid]
	return active, nil
}

func randomHandle() proto.HogXID {
	const n = 8
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (x *Service) Hoglock_signal(ctx context.Context, xid proto.HogXID, ns proto.HogNamespaces) error {

	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()

	// check a lease is present
	_, ok := x.leases[xid]
	if !ok {
		return fmt.Errorf("no lease")
	}

	signalled := x.signalled[xid] // isolate per-xid namespaces
	if signalled == nil {
		signalled = map[proto.HogNamespace]struct{}{}
		x.signalled[xid] = signalled
	}

	for _, ns := range ns {
		signalled[ns] = struct{}{}
	}

	return nil
}

func (x *Service) Hoglock_wait(ctx context.Context, xid proto.HogXID, prefix proto.HogNamespaces) error {

	x.lk.Lock()
	defer x.lk.Unlock()

	for {
		if x.isSignalled(xid, prefix) {
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

func (x *Service) isSignalled(xid proto.HogXID, prefix proto.HogNamespaces) bool {
	uncovered := map[proto.HogNamespace]struct{}{}
	for _, prefix := range prefix {
		uncovered[prefix] = struct{}{}
	}
	for ns := range x.signalled[xid] {
		for u := range uncovered {
			if strings.HasPrefix(ns, u) {
				delete(uncovered, u)
			}
		}
	}
	return len(uncovered) == 0
}
