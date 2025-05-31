package quic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/groundhogchain/chuckd/lib/must"
	"github.com/quic-go/quic-go"
)

// QUICNet implements the libnet.Network interface using QUIC
type QUICNet struct {
	mu        sync.RWMutex
	dialed    map[string]*quicConn     // address -> connection
	listeners map[string]*quicListener // address -> listener
	tlsConfig *tls.Config
}

// NewQUICNet creates a new QUIC network instance
func NewQUICNet(ctx context.Context) *QUICNet {

	// Generate a self-signed certificate for testing
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	must.NoError(ctx, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 180), // 180 days
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	must.NoError(ctx, err)

	cert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}

	return &QUICNet{
		dialed:    make(map[string]*quicConn),
		listeners: make(map[string]*quicListener),
		tlsConfig: &tls.Config{
			Certificates:       []tls.Certificate{*cert},
			InsecureSkipVerify: true, // For testing only
			NextProtos:         []string{"groundhog-quic"},
			ClientAuth:         tls.NoClientCert,
		},
	}
}

func (n *QUICNet) String() string {
	return "quic"
}

type quicListener struct {
	listener *quic.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	net      *QUICNet
	address  string

	mu      sync.Mutex
	conns   map[quic.Connection]struct{}
	streams chan acceptResult
	closed  bool
}

type acceptResult struct {
	stream quic.Stream
	conn   quic.Connection
	err    error
}

func (l *quicListener) handleConnection(conn quic.Connection) {
	for {
		stream, err := conn.AcceptStream(l.ctx)
		select {
		case l.streams <- acceptResult{stream: stream, conn: conn, err: err}:
			if err != nil {
				// On error, stop handling this connection
				l.mu.Lock()
				delete(l.conns, conn)
				l.mu.Unlock()
				return
			}
		case <-l.ctx.Done():
			return
		}
	}
}

func (l *quicListener) acceptLoop() {
	for {
		qconn, err := l.listener.Accept(l.ctx)
		if err != nil {
			// If we're closed or context is done, exit cleanly
			if l.ctx.Err() != nil || l.closed {
				return
			}
			// Otherwise try again
			continue
		}

		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			qconn.CloseWithError(0, "listener closed")
			return
		}
		l.conns[qconn] = struct{}{}
		l.mu.Unlock()

		// Start a goroutine to handle streams for this connection
		go l.handleConnection(qconn)
	}
}

func (l *quicListener) Accept() (net.Conn, error) {
	for {
		l.mu.Lock()
		if l.closed {
			l.mu.Unlock()
			return nil, net.ErrClosed
		}
		l.mu.Unlock()

		select {
		case result := <-l.streams:
			if result.err != nil {
				// Skip errors and continue to next stream
				continue
			}
			return &quicStream{
				Stream: result.stream,
				conn:   result.conn,
			}, nil
		case <-l.ctx.Done():
			return nil, l.ctx.Err()
		}
	}
}

func (l *quicListener) Close() error {

	// Remove ourselves from the QUICNet
	l.net.mu.Lock()
	delete(l.net.listeners, l.address)
	l.net.mu.Unlock()

	// are we calling close twice?
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true

	// Cancel context to stop all goroutines
	l.cancel()

	// Close all connections
	for conn := range l.conns {
		conn.CloseWithError(0, "listener closed")
	}
	l.conns = nil
	l.mu.Unlock()

	return l.listener.Close()
}

func (l *quicListener) Addr() net.Addr {
	return l.listener.Addr()
}

func (n *QUICNet) ListenAddress(ctx context.Context, address string) (net.Listener, error) {

	n.mu.Lock()
	if l, ok := n.listeners[address]; ok {
		n.mu.Unlock()
		return l, nil
	}

	listener, err := quic.ListenAddr(address, n.tlsConfig, &quic.Config{
		KeepAlivePeriod: time.Second * 30, // Default keep-alive period
	})
	if err != nil {
		n.mu.Unlock()
		return nil, err
	}

	// Create a new context with cancel function
	ctx, cancel := context.WithCancel(ctx)

	l := &quicListener{
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
		net:      n,
		address:  address,
		conns:    make(map[quic.Connection]struct{}),
		streams:  make(chan acceptResult, 10),
	}

	n.listeners[address] = l
	n.mu.Unlock()

	// Start the connection acceptor loop
	go l.acceptLoop()

	return l, nil
}

func (n *QUICNet) removeDialedConn(address string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.dialed, address)
}

func (n *QUICNet) addDialedConn(address string, conn *quicConn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.dialed[address] = conn
}

func (n *QUICNet) getDialedConn(address string) (*quicConn, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	conn, exists := n.dialed[address]
	return conn, exists
}

func (n *QUICNet) DialAddress(ctx context.Context, address string) (net.Conn, error) {
	// Try existing connection first
	if conn, exists := n.getDialedConn(address); exists {
		if stream, err := conn.openStream(ctx); err == nil {
			return stream, nil
		}
		// If we got an error, remove the connection and try again
		n.removeDialedConn(address)
	}

	// Create new connection
	qconn, err := quic.DialAddr(ctx, address, n.tlsConfig, &quic.Config{
		KeepAlivePeriod: time.Second * 30, // Default keep-alive period
	})
	if err != nil {
		return nil, err
	}

	conn := &quicConn{qconn: qconn}

	// Try to open a stream before adding the connection to our map
	stream, err := conn.openStream(ctx)
	if err != nil {
		qconn.CloseWithError(0, "failed to open initial stream")
		return nil, err
	}

	n.addDialedConn(address, conn)

	return stream, nil
}

type quicConn struct {
	qconn quic.Connection
	mu    sync.RWMutex
	err   error
}

func (c *quicConn) openStream(ctx context.Context) (net.Conn, error) {
	c.mu.RLock()
	if c.err != nil {
		c.mu.RUnlock()
		return nil, c.err
	}
	c.mu.RUnlock()

	stream, err := c.qconn.OpenStreamSync(ctx)
	if err != nil {
		c.mu.Lock()
		c.err = err
		c.mu.Unlock()
		return nil, err
	}

	return &quicStream{
		Stream: stream,
		conn:   c.qconn,
	}, nil
}

type quicStream struct {
	quic.Stream
	conn quic.Connection
}

func (s *quicStream) Close() error {
	return s.Stream.Close()
}

func (s *quicStream) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *quicStream) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}
