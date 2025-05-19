package testnet

import (
	"io"
	"net"
	"sync"
	"time"
)

type testConn struct {
	dialer   *testHalfConn
	acceptor *testHalfConn
}

func newTestConn(dialerAddr string, acceptorAddr string) *testConn {
	ad, da := make(chan []byte, 1), make(chan []byte, 1)
	return &testConn{
		dialer:   newTestHalfConn(dialerAddr, acceptorAddr, ad, da),
		acceptor: newTestHalfConn(acceptorAddr, dialerAddr, da, ad),
	}
}

type testHalfConn struct {
	localAddr  string
	remoteAddr string
	read       <-chan []byte
	write      chan<- []byte
	rlk        sync.Mutex
	unread     []byte
}

func newTestHalfConn(localAddr string, remoteAddr string, r <-chan []byte, w chan<- []byte) *testHalfConn {
	return &testHalfConn{
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
		read:       r,
		write:      w,
	}
}

func (x *testHalfConn) Read(b []byte) (n int, err error) {

	if len(b) == 0 {
		return 0, nil
	}

	x.rlk.Lock()
	defer x.rlk.Unlock()

	if len(x.unread) > 0 {
		return x.readUnread(b)
	}

	for len(x.unread) == 0 {
		var ok bool
		x.unread, ok = <-x.read

		if !ok {
			return 0, io.EOF
		}
	}

	return x.readUnread(b)
}

func (x *testHalfConn) readUnread(b []byte) (n int, err error) {
	n = min(len(x.unread), len(b))
	copy(b[:n], x.unread[:n])
	x.unread = x.unread[n:]
	return n, nil
}

func (x *testHalfConn) Write(b []byte) (n int, err error) {
	x.write <- b
	return len(b), nil
}

func (x *testHalfConn) Close() error {
	close(x.write)
	return nil
}

func (x *testHalfConn) LocalAddr() net.Addr {
	return newTestAddress(x.localAddr)
}

func (x *testHalfConn) RemoteAddr() net.Addr {
	return newTestAddress(x.remoteAddr)
}

func (x *testHalfConn) SetDeadline(t time.Time) error {
	return nil
}

func (x *testHalfConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (x *testHalfConn) SetWriteDeadline(t time.Time) error {
	return nil
}
