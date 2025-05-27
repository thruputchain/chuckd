package quic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"sync"
	"time"

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

func (x *QUICNet) DialAddress(ctx context.Context, address string) (net.Conn, error) {

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

	return streamConn{Stream: s, Remote: address}, nil
}

func (x *QUICNet) ListenAddress(ctx context.Context, address string) (net.Listener, error) {

	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"chuckd"},
	}

	listener, err := quic.ListenAddr(address, tlsConfig, nil)
	if err != nil {
		return nil, err
	}

	return streamListener{Context: ctx, Listener: listener, Local: address}, nil
}

func generateSelfSignedCert() (tls.Certificate, error) {

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Groundhog Chain"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:              []string{"localhost"},
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create tls.Certificate
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}

	return cert, nil
}
