package quic

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQUICNet(t *testing.T) {
	ctx := context.Background()
	net := NewQUICNet(ctx)

	// Start listener
	listener, err := net.ListenAddress(ctx, "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	// Get the actual address
	addr := listener.Addr().String()

	// Accept connections in a goroutine
	go func() {
		conn, err := listener.Accept()
		require.NoError(t, err)
		defer conn.Close()

		// Echo server
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		require.NoError(t, err)
		_, err = conn.Write(buf[:n])
		require.NoError(t, err)
	}()

	// Test dialing and connection reuse
	conn1, err := net.DialAddress(ctx, addr)
	require.NoError(t, err)
	defer conn1.Close()

	// Write data
	testData := []byte("hello world")
	_, err = conn1.Write(testData)
	require.NoError(t, err)

	// Read response
	buf := make([]byte, len(testData))
	_, err = io.ReadFull(conn1, buf)
	require.NoError(t, err)
	require.Equal(t, testData, buf)

	// Test connection reuse
	conn2, err := net.DialAddress(ctx, addr)
	require.NoError(t, err)
	defer conn2.Close()

	// Verify it's using the same underlying QUIC connection
	require.Equal(t, conn1.(*quicStream).conn.LocalAddr(), conn2.(*quicStream).conn.LocalAddr())
}

func TestQUICNetConcurrent(t *testing.T) {
	ctx := context.Background()
	net := NewQUICNet(ctx)

	// Start listener
	listener, err := net.ListenAddress(ctx, "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	// Get the actual address
	addr := listener.Addr().String()

	// Accept connections in a goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c io.ReadWriteCloser) {
				defer c.Close()
				// Echo server
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				_, err = c.Write(buf[:n])
				if err != nil {
					return
				}
			}(conn)
		}
	}()

	// Test concurrent connections
	const numConns = 10
	done := make(chan struct{})

	for i := 0; i < numConns; i++ {
		go func() {
			defer func() { done <- struct{}{} }()

			conn, err := net.DialAddress(ctx, addr)
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()

			// Write data
			testData := []byte("hello world")
			_, err = conn.Write(testData)
			if err != nil {
				t.Error(err)
				return
			}

			// Read response
			buf := make([]byte, len(testData))
			_, err = io.ReadFull(conn, buf)
			if err != nil {
				t.Error(err)
				return
			}
			if string(buf) != string(testData) {
				t.Error("data mismatch")
				return
			}
		}()
	}

	// Wait for all connections to complete
	timeout := time.After(5 * time.Second)
	for i := 0; i < numConns; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("test timed out")
		}
	}
}
