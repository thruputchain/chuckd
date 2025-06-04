package testnet

import (
	"context"
	"fmt"
	"net"
	"sync"
)

type TestNetwork struct {
	lk        sync.Mutex
	listeners map[string]*testListener // address listening on -> listener
	conns     map[*testConn]bool
}

func NewTestNetwork() *TestNetwork {
	return &TestNetwork{
		listeners: map[string]*testListener{},
		conns:     map[*testConn]bool{},
	}
}

func (x *TestNetwork) String() string {
	return "testnet"
}

func (x *TestNetwork) DialAddress(ctx context.Context, address string) (net.Conn, error) {
	return x.Dial(ctx, "", address)
}

func (x *TestNetwork) Dial(ctx context.Context, network string, address string) (net.Conn, error) {
	x.lk.Lock()
	defer x.lk.Unlock()

	l, ok := x.listeners[address]
	if !ok {
		return nil, fmt.Errorf("no listener on address %v", address)
	}

	return l.dial(), nil
}

func (x *TestNetwork) Listen(ctx context.Context, network string, address string) (net.Listener, error) {
	return x.ListenAddress(ctx, address)
}

func (x *TestNetwork) ListenAddress(ctx context.Context, address string) (net.Listener, error) {
	x.lk.Lock()
	defer x.lk.Unlock()

	if _, ok := x.listeners[address]; ok {
		return nil, fmt.Errorf("listener address taken")
	}

	l := newTestListener(x, address)
	x.listeners[address] = l

	return l, nil
}
