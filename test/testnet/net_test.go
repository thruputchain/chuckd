package testnet

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
)

func TestDialAccept(t *testing.T) {

	ctx := context.Background()
	testAddress := "alpha"
	testPkt := []byte{1, 2, 3}

	v := NewTestNetwork()
	l, _ := v.listen(ctx, "", testAddress)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		c, _ := l.Accept()
		b := make([]byte, 3)
		n, err := c.Read(b)
		if n != 3 {
			t.Errorf("expecting 3, got %d", n)
		}
		if err != nil {
			t.Errorf("expecting no error (%s)", err)
		}
		if bytes.Compare(testPkt, b) != 0 {
			t.Errorf("packets unequal")
		}
		c.Close()
	}()

	c, _ := v.dial(ctx, "", testAddress)
	n, err := c.Write(testPkt)
	if n != 3 {
		t.Errorf("expecting 3, got %d", n)
	}
	if err != nil {
		t.Errorf("expecting no error (%s)", err)
	}
	c.Close()

	wg.Wait()
}

func TestCloseRemovesListenerAddress(t *testing.T) {
	ctx := context.Background()
	testAddress := "alpha"

	v := NewTestNetwork()
	l, err := v.listen(ctx, "", testAddress)
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	if _, err := v.dial(ctx, "", testAddress); err == nil {
		t.Fatalf("expected dial to fail after listener close")
	} else if !strings.Contains(err.Error(), "no listener on address") {
		t.Fatalf("unexpected dial error: %v", err)
	}

	if _, err := v.listen(ctx, "", testAddress); err != nil {
		t.Fatalf("expected address to be reusable after close: %v", err)
	}
}
