package testnet

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type testListener struct {
	net     *TestNetwork
	addr    string
	ch      chan *testHalfConn
	lk      sync.Mutex
	closed  bool
	naccept int
}

func newTestListener(n *TestNetwork, address string) *testListener {
	return &testListener{
		net:  n,
		addr: address,
		ch:   make(chan *testHalfConn, 1),
	}
}

// dial is called inside a network lock!
func (x *testListener) dial() (net.Conn, error) {
	x.lk.Lock()
	defer x.lk.Unlock()
	if x.closed {
		return nil, net.ErrClosed
	}

	c := newTestConn(
		fmt.Sprintf("%s/dialer/%d", x.addr, x.naccept),
		fmt.Sprintf("%s/listener/%d", x.addr, x.naccept),
	)
	x.naccept += 1

	x.ch <- c.acceptor

	return c.dialer, nil
}

func (x *testListener) Accept() (net.Conn, error) {
	c, ok := <-x.ch
	if !ok {
		return nil, io.EOF
	}
	return c, nil
}

func (x *testListener) Close() error {
	x.lk.Lock()
	defer x.lk.Unlock()
	if x.closed {
		return nil
	}
	x.closed = true
	close(x.ch)
	return nil
}

func (x *testListener) Addr() net.Addr {
	return newTestAddress(x.addr)
}
