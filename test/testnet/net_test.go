package testnet

import (
	"bytes"
	"sync"
	"testing"
)

func TestDialAccept(t *testing.T) {

	testAddress := "alpha"
	testPkt := []byte{1, 2, 3}

	v := NewTestNetwork()
	l, _ := v.Listen("", testAddress)

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

	c, _ := v.Dial("", testAddress)
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
