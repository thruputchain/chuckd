package hoglock

import (
	"context"
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
	inventory []string //XXX
	//
	lk   sync.Mutex
	cond *sync.Cond
	//
	demand  map[proto.HogHost]int
	leases  map[proto.HogXID]*lease
	locks   map[proto.HogHost]proto.HogXID
	signals map[proto.HogXID]map[proto.HogNamespace]struct{}
}

type lease struct {
	renewed time.Time
	hosts   proto.HogHosts
}

func NewService(allHosts []string) *Service {
	x := &Service{
		inventory: allHosts,
		demand:    map[proto.HogHost]int{},
		locks:     map[proto.HogHost]proto.HogXID{},
		leases:    map[proto.HogXID]*lease{},
		//
		signals: map[proto.HogXID]map[proto.HogNamespace]struct{}{},
	}
	x.cond = sync.NewCond(&x.lk)
	for _, h := range allHosts {
		x.demand[h] = 0
	}
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

func (x *Service) acquireLock(ctx context.Context, xid proto.HogXID, host proto.HogHost) error {
	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if _, locked := x.locks[host]; locked {
			x.cond.Wait()
		} else {
			x.locks[host] = xid
			return nil
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

	delete(x.signals, xid)

	lease := x.leases[xid]
	delete(x.leases, xid)

	slices.Reverse(lease.hosts)
	liblog.Infof("releasing locks %v, while expiring lease %v", lease.hosts, xid)

	for _, h := range lease.hosts {
		delete(x.locks, h)
	}
	x.changeDemand(-1, lease.hosts...)
}

func (x *Service) pickNHosts(_ context.Context, n int, excluding proto.HogHosts) ([]proto.HogHost, error) {
	x.lk.Lock()
	defer x.lk.Unlock()

	wh := WeightedHostsFromMap(x.demand, excluding)
	if len(wh) < n {
		return nil, fmt.Errorf("not enough available hosts")
	}
	return wh[:n].ToHosts(), nil
}

func (x *Service) changeDemand(delta int, hosts ...proto.HogHost) {
	for _, h := range hosts {
		x.demand[h] += delta
	}
}

func (x *Service) lockChangeDemand(delta int, hosts ...proto.HogHost) {
	x.lk.Lock()
	defer x.lk.Unlock()
	x.changeDemand(delta, hosts...)
}

// --- rpc ---

func (x *Service) Hoglock_null(context.Context) error {
	return nil
}

func (x *Service) Hoglock_lock(
	ctx context.Context,
	xid proto.HogXID,
	ask proto.HogAsk,
) (assigned proto.HogHosts, err error) {

	liblog.Infof("hoglock_lock %v and %v", ask.Fixed, ask.Variable)
	defer func() {
		if err != nil {
			liblog.Errorf("hoglock_lock for %v and %v not acquired (%v)", ask.Fixed, ask.Variable, err)
		} else {
			liblog.Infof("hoglock_lock for %v and %v acquired under lease %v", ask.Fixed, ask.Variable, xid)
		}
	}()

	if len(ask.Fixed)+int(ask.Variable) == 0 {
		return nil, nil
	}

	assigned, err = x.pickNHosts(ctx, int(ask.Variable), ask.Fixed)
	if err != nil {
		return nil, err
	}

	// all = fixed + assigned
	all := make(proto.HogHosts, 0, len(ask.Fixed)+int(ask.Variable))
	for _, f := range ask.Fixed {
		all = append(all, f)
	}
	for _, a := range assigned {
		all = append(all, a)
	}

	x.lockChangeDemand(+1, all...) // demand is reverted on lock release

	sort.Strings(all)
	acquired := proto.HogHosts{}
	var errTotal error
	for _, h := range all {
		err = x.acquireLock(ctx, xid, h)
		// if a lock is acquired, the expiration routine will revert its demand,
		// otherwise we need to revert it here.
		if err != nil {
			x.lockChangeDemand(-1, h)
			errTotal = err
		} else {
			acquired = append(acquired, h)
		}
	}

	x.lk.Lock()
	defer x.lk.Unlock()

	x.leases[xid] = &lease{renewed: time.Now(), hosts: acquired}

	if errTotal != nil {
		return nil, errTotal
	}

	return assigned, nil
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

func (x *Service) Hoglock_signal(ctx context.Context, xid proto.HogXID, ns proto.HogNamespaces) error {

	x.lk.Lock()
	defer x.lk.Unlock()
	defer x.cond.Broadcast()

	// check a lease is present
	_, ok := x.leases[xid]
	if !ok {
		return fmt.Errorf("no lease")
	}

	signalled := x.signals[xid] // isolate per-xid namespaces
	if signalled == nil {
		signalled = map[proto.HogNamespace]struct{}{}
		x.signals[xid] = signalled
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
	for ns := range x.signals[xid] {
		for u := range uncovered {
			if strings.HasPrefix(ns, u) {
				delete(uncovered, u)
			}
		}
	}
	return len(uncovered) == 0
}
