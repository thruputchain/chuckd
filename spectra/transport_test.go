package spectra

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/groundhogchain/chuckd/broadcast"
	"github.com/groundhogchain/chuckd/lib/log"
	"github.com/groundhogchain/chuckd/runtime"
	proto "github.com/groundhogchain/protocol-go"
)

func TestMain(m *testing.M) {
	log.LogVerbosely()
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts()
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestRoundtrip(t *testing.T) {

	bcfg := proto.BroadcastConfig{
		Multicast_tree_ttl_msec: 1e3,
		Chunk_size:              38,
	}

	tn := broadcast.NewBroadcast3TCP(bcfg)
	tn.Start()

	scfg := proto.SpectraConfig{
		Num_payload_channels: 2,
		Num_parity_channels:  1,
		Partial_ttl_msec:     3e3,
	}
	rand0 := runtime.NewRandFromRand(tn.Rand[0])
	rand1 := runtime.NewRandFromRand(tn.Rand[1])
	rand2 := runtime.NewRandFromRand(tn.Rand[2])

	s0 := NewSpectraTransport(tn.Ctx, scfg, rand0, tn.B[0].BulkAPI())
	s1 := NewSpectraTransport(tn.Ctx, scfg, rand1, tn.B[1].BulkAPI())
	s2 := NewSpectraTransport(tn.Ctx, scfg, rand2, tn.B[2].BulkAPI())

	payload := []byte{1, 2, 3, 4, 5, 6}

	go func() {
		_, err := s0.BulkSend(tn.Ctx, payload)
		if err != nil {
			t.Error(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _, p, err := s1.BulkReceive(tn.Ctx)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(p, payload) != 0 {
			t.Errorf("unexpected payload")
		}
	}()

	go func() {
		defer wg.Done()
		_, _, p, err := s2.BulkReceive(tn.Ctx)
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(p, payload) != 0 {
			t.Errorf("unexpected payload")
		}
	}()

	wg.Wait()
}
