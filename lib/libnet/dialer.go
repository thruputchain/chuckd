package libnet

import "net"

type Network interface {
	AddressListener
	AddressDialer
}

type AddressListener interface {
	ListenAddress(address string) (net.Listener, error)
}

type AddressDialer interface {
	DialAddress(address string) (net.Conn, error)
}

type TCPNet struct{}

func (TCPNet) ListenAddress(address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}

func (TCPNet) DialAddress(address string) (net.Conn, error) {
	return net.Dial("tcp", address)
}
