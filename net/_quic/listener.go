package quic

import (
	"context"
	"net"

	"github.com/quic-go/quic-go"
)

type streamListener struct {
	context.Context
	quic.Listener
	Local string
}

func (x streamListener) Accept() (net.Conn, error) {

	conn, err := x.Listener.Accept(x.Context)
	if err != nil {
		return nil, err
	}

	

	return streamListener{}
}

func (x streamListener) Addr() net.Addr {
	return address(x.Local)
}
