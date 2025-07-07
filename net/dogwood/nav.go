package dogwood

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/form"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/groundhogchain/chuckd/net/router"
	"github.com/groundhogchain/chuckd/util/dogwoodutil"
	"github.com/groundhogchain/chuckd/util/xdrutil"
	proto "github.com/groundhogchain/protocol-go"
)

// Navigator maintains a cache of routes rooted at this node that have been socialized with others,
// and can be used for broadcasting. It also maintains a cache of trees socialized by others.
type Navigator struct {
	// systems
	time        libtime.Time
	rand        *rand.Rand
	disseminate DisseminateRoutesFn
	router      router.DogwoodRouter
	// params
	self    proto.ValidatorID
	ttLive  time.Duration // ttl for a reusable route
	ttUse   time.Duration // max duration of route in use, i.e. max duration of dissemination
	nRoutes int           // number of reusable routes at any time
	// state
	lk     sync.Mutex
	ours   map[proto.DogwoodTreeHash]*stampedRoute
	theirs map[proto.DogwoodTreeHash]*stampedRoute
	// other
	marshaler xdrutil.XdrMarshaler
}

type DisseminateRoutesFn func([]*dogwoodutil.LinkedTreeNode)

type stampedRoute struct {
	Stamp time.Time
	Tree  *dogwoodutil.LinkedTreeNode
}

func NewNavigator(
	ctx context.Context,
	time libtime.Time,
	rand *rand.Rand,
	disseminate DisseminateRoutesFn,
	router router.DogwoodRouter,
	self proto.ValidatorID,
	ttLive time.Duration, // route ttl
	ttUse time.Duration, // max duration of route in use, i.e. max duration of dissemination
	nRoutes int, // number of available routes at any time
) *Navigator {
	return &Navigator{
		time:        time,
		rand:        rand,
		disseminate: disseminate,
		router:      router,
		self:        self,
		ttLive:      ttLive,
		ttUse:       ttUse,
		nRoutes:     nRoutes,
		ours:        map[proto.DogwoodTreeHash]*stampedRoute{},
		theirs:      map[proto.DogwoodTreeHash]*stampedRoute{},
		marshaler:   xdrutil.NewXdrMarshaler(),
	}
}

func (nav *Navigator) Start(ctx context.Context) {
	go nav.refill(ctx)
	go nav.expire(ctx)
}

type NavRoute struct {
	Hash       proto.DogwoodTreeHash // route hash
	Downstream []proto.ValidatorID   // downstream validators on route
}

func (nav *Navigator) CacheRoute(ctx context.Context, t *dogwoodutil.LinkedTreeNode) {

	nav.lk.Lock()
	defer nav.lk.Unlock()

	nav.theirs[dogwoodutil.TreeHash(nav.marshaler, t)] = &stampedRoute{Stamp: nav.time.Now(), Tree: t}
}

func (nav *Navigator) LookupRoute(ctx context.Context, h proto.DogwoodTreeHash) (NavRoute, error) {

	nav.lk.Lock()
	defer nav.lk.Unlock()

	x, ok := nav.theirs[h]
	if ok {
		d, _ := x.Tree.FindChildren(nav.self)
		return NavRoute{Hash: h, Downstream: d}, nil
	}

	x, ok = nav.ours[h]
	if ok {
		d, found := x.Tree.FindChildren(nav.self)
		must.Assert(ctx, found)
		return NavRoute{Hash: h, Downstream: d}, nil
	}

	liblog.Errorf("route %v not found", form.Display(h))

	return NavRoute{}, fmt.Errorf("route not found")
}

func (nav *Navigator) GetRoutes(ctx context.Context, n int) ([]NavRoute, error) {

	nav.lk.Lock()
	defer nav.lk.Unlock()

	now := nav.time.Now()
	routes := []NavRoute{}
	for h, sr := range nav.ours { // ranging over a map randomizes the route choices
		if len(routes) >= n {
			break
		}
		// filter routes that are too old (use routes no older than stamp+ttLive)
		if now.Sub(sr.Stamp) > nav.ttLive {
			continue
		}
		downstream, ok := sr.Tree.FindChildren(nav.self)
		must.Assert(ctx, ok)
		r := NavRoute{Hash: h, Downstream: downstream}
		routes = append(routes, r)
	}

	if n > 0 && len(routes) == 0 {
		return nil, fmt.Errorf("no available routes")
	}

	return routes, nil
}

func (nav *Navigator) refill(_ context.Context) {
	tick := nav.ttLive / 3
	for {
		nav.time.Sleep(tick)
		now := nav.time.Now()
		next := now.Add(2 * tick)

		nav.lk.Lock()
		// count number of routes which will still be live on the next iteration
		nDying := 0
		for _, sr := range nav.ours {
			eol := sr.Stamp.Add(nav.ttLive) // end of life for route; it won't be used after
			if next.Compare(eol) >= 0 {
				nDying += 1
			}
		}
		nav.lk.Unlock()

		// replenish routes
		m := max(0, nav.nRoutes-nDying)
		rs := make([]*dogwoodutil.LinkedTreeNode, m)
		for i := 0; i < m; i++ {
			rs[m] = nav.router.BuildDogwoodTree(nav.self)
		}
		nav.disseminate(rs)

		if len(rs) > 0 {
			nav.lk.Lock()
			for _, t := range rs {
				h := dogwoodutil.TreeHash(nav.marshaler, t)
				nav.ours[h] = &stampedRoute{Stamp: now, Tree: t}
			}
			nav.lk.Unlock()
		}
	}
}

func (nav *Navigator) expire(_ context.Context) {
	for {
		nav.time.Sleep(nav.ttLive)
		now := nav.time.Now()
		nav.lk.Lock()
		for key, sr := range nav.ours {
			if now.Sub(sr.Stamp) > nav.ttLive+nav.ttUse {
				delete(nav.ours, key)
			}
		}
		if len(nav.ours) == 0 {
			liblog.Errorf("no navigator routes left")
		}
		for key, sr := range nav.theirs {
			if now.Sub(sr.Stamp) > nav.ttLive+nav.ttUse {
				delete(nav.theirs, key)
			}
		}
		nav.lk.Unlock()
	}
}
