package broadcast

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/runtime"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	liblog.LogLevel(zerolog.DebugLevel)
	runtime.SetRootSeed(0)
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestStreaming(t *testing.T) {

	tn := NewBroadcast3Testnet(DefaultBroadcastConfig)
	tn.Start()

	payload := []byte{1, 2, 3}

	go func() {
		sch := tn.B[0].StreamingAPI().MakeChannel(tn.Ctx)
		sch.Send(tn.Ctx, payload)
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		rch := tn.B[1].StreamingAPI().AcceptChannel()
		_, _, p, err := rch.Receive(tn.Ctx)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(p, payload) {
			t.Errorf("unexpected payload")
		}
	}()

	go func() {
		defer wg.Done()
		rch := tn.B[2].StreamingAPI().AcceptChannel()
		_, _, p, err := rch.Receive(tn.Ctx)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(p, payload) {
			t.Errorf("unexpected payload")
		}
	}()

	wg.Wait()
}

func TestBulk(t *testing.T) {

	cfg := proto.BroadcastConfig{
		Multicast_tree_ttl_msec: 1e3,
		Chunk_size:              1,
	}

	tn := NewBroadcast3Testnet(cfg)
	tn.Start()

	payload := make([]byte, 39)
	for i := range payload {
		payload[i] = byte(i)
	}

	go func() {
		_, err := tn.B[0].BulkAPI().BulkSend(tn.Ctx, payload)
		if err != nil {
			t.Error(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, p, err := tn.B[1].BulkAPI().BulkReceive(tn.Ctx)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(p, payload) {
			t.Errorf("unexpected payload")
		}
	}()

	go func() {
		defer wg.Done()
		_, p, err := tn.B[2].BulkAPI().BulkReceive(tn.Ctx)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(p, payload) {
			t.Errorf("unexpected payload")
		}
	}()

	wg.Wait()
}
