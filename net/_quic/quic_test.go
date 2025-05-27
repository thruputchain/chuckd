package quic

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/quic-go/quic-go"
)

func _TestQuic(t *testing.T) {

	// quic server
	go func() {
		ctx := context.Background()
		addr, err := net.ResolveUDPAddr("udp", ":9111")
		must.NoError(ctx, err)

		conn, err := net.ListenUDP("udp", addr)
		must.NoError(ctx, err)

		tr := &quic.Transport{
			Conn: conn,
		}

		tlsConf := &tls.Config{}   //XXX
		quicConf := &quic.Config{} //XXX
		ln, err := tr.Listen(tlsConf, quicConf)
		must.NoError(ctx, err)

		for {
			conn, err := ln.Accept(ctx)
			must.NoError(ctx, err)
			go func() {
				for {
					str, err := conn.AcceptStream(ctx)
					must.NoError(ctx, err)
					r, err := io.ReadAll(str)
					must.NoError(ctx, err)
					fmt.Println("received", string(r))
				}
			}()
		}
	}()

	// quic client
	go func() {
	}()

	select {}
}
