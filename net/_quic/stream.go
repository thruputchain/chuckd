package quic

import (
	"net"

	"github.com/quic-go/quic-go"
)

type address string

func (a address) Network() string {
	return "QUIC"
}

func (a address) String() string {
	return string(a)
}

type streamConn struct {
	quic.Stream
	Remote string
}

func (x streamConn) LocalAddr() net.Addr {
	return address(x.Stream.StreamID())
}

func (x streamConn) RemoteAddr() net.Addr {
	return address(x.Remote)
}
