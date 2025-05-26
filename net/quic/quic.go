package quic

import (
	"context"
	"crypto/tls"
	"net"
	"sync"

	"github.com/groundhogchain/chuckd/lib/libnet"
	"github.com/quic-go/quic-go"
)

type QUICNet struct {
	ctx   context.Context
	lk    sync.Mutex
	conns map[string]quic.Connection
}

func NewQUICNet(ctx context.Context) *QUICNet {
	return &QUICNet{
		ctx:   ctx,
		conns: map[string]quic.Connection{},
	}
}

func (x *QUICNet) String() string {
	return "QUIC"
}

func (x *QUICNet) DialAddress(ctx context.Context, address string) (libnet.Conn, error) {

	x.lk.Lock()
	cached, ok := x.conns[address]
	x.lk.Unlock()

	var err error
	if !ok {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"chuckd"},
		}
		cached, err = quic.DialAddr(ctx, address, tlsConfig, nil)
		if err != nil {
			return nil, err
		}
		x.conns[address] = cached
	}

	s, err := cached.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (x *QUICNet) ListenAddress(ctx context.Context, address string) (net.Listener, error) {
	panic("XXX")
}
