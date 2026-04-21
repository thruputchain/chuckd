package dogwood

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/liblog"
	"github.com/groundhogchain/lib-go/libstat"
	"github.com/groundhogchain/lib-go/libtime"
	"github.com/groundhogchain/lib-go/libxdr"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

// Navigator maintains a cache of routes rooted at this node that have been socialized with others,
// and can be used for broadcasting. It also maintains a cache of trees socialized by others.
type Navigator struct {
	// systems
	time         libtime.Time
	rand         *rand.Rand
	disseminator RouteDisseminator
	router       router.Router
	// params
	self            proto.ValidatorID
	routeTTL        time.Duration
	routeMaxLatency time.Duration // maximum broadcast latency
	nRoutes         int           // number of reusable routes at any time
	// state
	lk     sync.Mutex
	ours   map[proto.DogwoodTreeHash]*stampedRoute
	theirs map[proto.DogwoodTreeHash]*stampedRoute
	// perf
	marshaler libxdr.XdrMarshaler
	// signaling
	green sync.Once
	ready chan struct{}
	// meters
	routeNotFound libstat.Counter
	routeFound    libstat.Counter
}

type RouteDisseminator interface {
	DisseminateRoutes(context.Context, []*dogwoodutil.LinkedTreeNode)
}

type stampedRoute struct {
	Stamp time.Time
	Tree  *dogwoodutil.LinkedTreeNode
}

func NewNavigator(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	router router.Router,
	routeTTL time.Duration, // route ttl
	routeMaxLatency time.Duration, // max duration of propagation along a route
	nRoutes int, // number of available routes at any time
) *Navigator {
	return &Navigator{
		time:            time,
		rand:            rand,
		router:          router,
		self:            protoutil.PublicVID(router.Self()),
		routeTTL:        routeTTL,
		routeMaxLatency: routeMaxLatency,
		nRoutes:         nRoutes,
		ours:            map[proto.DogwoodTreeHash]*stampedRoute{},
		theirs:          map[proto.DogwoodTreeHash]*stampedRoute{},
		marshaler:       libxdr.NewXdrMarshaler(),
		ready:           make(chan struct{}),
	}
}

func (nav *Navigator) Start(ctx context.Context, disseminator RouteDisseminator) {
	nav.disseminator = disseminator
	go nav.refill(ctx)
	go nav.advertise(ctx)
	go nav.expire(ctx)
}

func (nav *Navigator) WaitReady(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-nav.ready:
		return nil
	}
}

type NavRoute struct {
	Hash       proto.DogwoodTreeHash // route hash
	Downstream []proto.ValidatorID   // downstream validators on route
}

// GenRoute builds a new route, which is not cached and not disseminated
func (nav *Navigator) GenRoute(ctx context.Context) *dogwoodutil.LinkedTreeNode {
	return nav.router.BuildDogwoodTree(nav.self)
}

func (nav *Navigator) CacheRoute(ctx context.Context, t *dogwoodutil.LinkedTreeNode) {

	nav.lk.Lock()
	defer nav.lk.Unlock()

	h := dogwoodutil.TreeHash(nav.marshaler, t)
	if _, ok := nav.theirs[h]; ok {
		return
	}

	nav.theirs[h] = &stampedRoute{Stamp: nav.time.Now(), Tree: t}
}

func (nav *Navigator) LookupRoute(ctx context.Context, h proto.DogwoodTreeHash) (NavRoute, error) {

	nav.lk.Lock()
	defer nav.lk.Unlock()

	x, ok := nav.theirs[h]
	if ok {
		nav.routeFound.Add(1)
		d, _ := x.Tree.FindChildren(nav.self)
		return NavRoute{Hash: h, Downstream: d}, nil
	}

	x, ok = nav.ours[h]
	if ok {
		nav.routeFound.Add(1)
		d, found := x.Tree.FindChildren(nav.self)
		must.Assert(ctx, found)
		return NavRoute{Hash: h, Downstream: d}, nil
	}

	nav.routeNotFound.Add(1)

	return NavRoute{}, fmt.Errorf("route not found")
}

// XXX: return non-overlapping routes across multiple invokations of GetRoute(1)
func (nav *Navigator) GetRoute(ctx context.Context) (NavRoute, error) {

	nav.lk.Lock()
	defer nav.lk.Unlock()

	now := nav.time.Now()
	for h, sr := range nav.ours { // ranging over a map randomizes the route choice
		// skip routes that are too old (use routes no older than stamp+ttLive)
		if now.Sub(sr.Stamp) > nav.routeTTL-nav.routeMaxLatency {
			continue
		}
		downstream, ok := sr.Tree.FindChildren(nav.self)
		must.Assert(ctx, ok)
		return NavRoute{Hash: h, Downstream: downstream}, nil
	}

	return NavRoute{}, fmt.Errorf("no available routes")
}

func (nav *Navigator) replenish(ctx context.Context, m int) {

	if m == 0 {
		return
	}

	rs := make([]*dogwoodutil.LinkedTreeNode, m)
	for i := 0; i < m; i++ {
		rs[i] = nav.router.BuildDogwoodTree(nav.self)
	}

	nav.disseminator.DisseminateRoutes(ctx, rs)
	nav.time.Sleep(nav.routeMaxLatency) // wait for routes to propagate

	now := nav.time.Now()

	nav.lk.Lock()
	for _, t := range rs {
		h := dogwoodutil.TreeHash(nav.marshaler, t)
		nav.ours[h] = &stampedRoute{Stamp: now, Tree: t}
	}
	nav.lk.Unlock()

	nav.green.Do(func() { close(nav.ready) })
}

func (nav *Navigator) refill(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		nav.replenish(ctx, nav.nRoutes)
		nav.time.Sleep(nav.routeTTL / 2)
	}
}

func (nav *Navigator) advertise(ctx context.Context) {
	for {
		nav.time.Sleep(nav.routeMaxLatency * 7)

		select {
		case <-ctx.Done():
			return
		default:
		}

		now := nav.time.Now()

		// pick routes eligible to readvertise
		nav.lk.Lock()
		rs := make([]*dogwoodutil.LinkedTreeNode, 0)
		for _, v := range nav.ours {
			if now.Sub(v.Stamp) < nav.routeTTL-2*nav.routeMaxLatency {
				rs = append(rs, v.Tree)
			}
		}
		nav.lk.Unlock()

		nav.disseminator.DisseminateRoutes(ctx, rs)
		nav.time.Sleep(nav.routeMaxLatency) // wait for routes to propagate

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (nav *Navigator) expire(ctx context.Context) {
	for {
		nav.time.Sleep(min(nav.routeTTL, 2*nav.routeMaxLatency))

		select {
		case <-ctx.Done():
			return
		default:
		}

		now := nav.time.Now()

		nav.lk.Lock()
		for key, sr := range nav.ours {
			if now.Sub(sr.Stamp) > nav.routeTTL-2*nav.routeMaxLatency {
				delete(nav.ours, key)
			}
		}
		if len(nav.ours) == 0 {
			liblog.Errorf(ctx, "no navigator routes left")
		}
		for key, sr := range nav.theirs {
			if now.Sub(sr.Stamp) > nav.routeTTL+nav.routeMaxLatency {
				delete(nav.theirs, key)
			}
		}
		nav.lk.Unlock()
	}
}

// stats

type NavigatorStats struct {
	RouteFound    libstat.CounterStats
	RouteNotFound libstat.CounterStats
}

func (x NavigatorStats) Summarize() string {
	return fmt.Sprintf("routes-found: %d, routes-not-found: %d", x.RouteFound.Count, x.RouteNotFound.Count)
}

func (nav *Navigator) Stats() NavigatorStats {
	return NavigatorStats{
		RouteFound:    nav.routeFound.Stats(),
		RouteNotFound: nav.routeNotFound.Stats(),
	}
}
