package libnet

import "net"

type AddressDialer interface {
	DialAddress(address string) (net.Conn, error)
}

type TCPAddressDialer struct{}

func (TCPAddressDialer) DialAddress(address string) (net.Conn, error) {
	return net.Dial("tcp", address)
}
