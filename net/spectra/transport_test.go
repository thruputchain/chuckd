package spectra

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/net/broadcast"
	"github.com/groundhogchain/chuckd/runtime"
	proto "github.com/groundhogchain/protocol-go"
)

func TestMain(m *testing.M) {
	liblog.LogVerbosely()
	runtime.SetRootSeed(0)
	runtime.CaptureInterrupts()
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestRoundtripAligned(t *testing.T) {
	scfg := proto.SpectraConfig{
		Num_payload_channels: 2,
		Num_parity_channels:  1,
		Partial_ttl_msec:     3e3,
	}
	payload := []byte{1, 2, 3, 4, 5, 6}
	testRoundtrip(t, scfg, payload)
}

func TestRoundtripMisaligned(t *testing.T) {
	scfg := proto.SpectraConfig{
		Num_payload_channels: 2,
		Num_parity_channels:  1,
		Partial_ttl_msec:     3e3,
	}
	payload := []byte{1, 2, 3, 4, 5}
	testRoundtrip(t, scfg, payload)
}

func TestRoundtripNoRedundancy(t *testing.T) {
	scfg := proto.SpectraConfig{
		Num_payload_channels: 2,
		Num_parity_channels:  0,
		Partial_ttl_msec:     3e3,
	}
	payload := []byte{1, 2, 3, 4, 5}
	testRoundtrip(t, scfg, payload)
}

func TestRoundtripTooManyChannels(t *testing.T) {
	scfg := proto.SpectraConfig{
		Num_payload_channels: 3,
		Num_parity_channels:  1,
		Partial_ttl_msec:     3e3,
	}
	payload := []byte{1, 2}
	testRoundtrip(t, scfg, payload)
}

func testRoundtrip(t *testing.T, scfg proto.SpectraConfig, payload []byte) {

	bcfg := proto.BroadcastConfig{
		Multicast_tree_ttl_msec: 1e3,
		Chunk_size:              1,
	}

	tn := broadcast.NewBroadcast3Testnet(bcfg)
	tn.Start()

	rand0 := runtime.NewRandFromRand(tn.Rand[0])
	rand1 := runtime.NewRandFromRand(tn.Rand[1])
	rand2 := runtime.NewRandFromRand(tn.Rand[2])

	s0 := NewSpectraTransport(tn.Ctx, scfg, rand0, tn.B[0].BulkAPI())
	s1 := NewSpectraTransport(tn.Ctx, scfg, rand1, tn.B[1].BulkAPI())
	s2 := NewSpectraTransport(tn.Ctx, scfg, rand2, tn.B[2].BulkAPI())

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
