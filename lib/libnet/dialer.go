package libnet

import (
	"context"
	"io"
	"net"
)

type Network interface {
	AddressListener
	AddressDialer
	String() string
}

type Conn interface {
	io.Reader
	io.Writer
	io.Closer
}

type AddressListener interface {
	ListenAddress(ctx context.Context, address string) (net.Listener, error)
}

type AddressDialer interface {
	DialAddress(ctx context.Context, address string) (net.Conn, error)
}

type TCPNet struct{}

func (TCPNet) String() string {
	return "tcp"
}

func (TCPNet) ListenAddress(ctx context.Context, address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}

func (TCPNet) DialAddress(ctx context.Context, address string) (net.Conn, error) {
	return net.Dial("tcp", address)
}
