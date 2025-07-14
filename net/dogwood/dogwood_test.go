package dogwood

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
	"time"

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

func TestSendReceiveTransmission(t *testing.T) {

	ctx := context.Background()
	cfg := proto.DogwoodConfig{
		Route_count:             7,
		Route_time_to_live_msec: 60e3, // 1 minute
		Route_time_to_use_msec:  200,  // 200 msec
		//
		Endpoint_queue_length:      20,
		Endpoint_idle_ttl_msec:     60e3, // 1 minute
		Endpoint_redial_delay_msec: 50,   // 50 msec
		//
		Stream_receiver_queue_length: 20,
		Stream_receiver_worker_count: 10,
		Stream_sender_worker_count:   10,
		//
		Transmission_receiver_worker_count: 10,
		Transmission_sender_worker_count:   10,
		//
		Transmission_config: &proto.TransmissionConfig{
			Num_payload_sheaves: 5,
			Num_parity_sheaves:  2,
			Max_chunk_bytes:     60e3, // 60K
			Partial_ttl_msec:    1e3,  // 1 sec
		},
	}

	tn := NewDogwood3Testnet(cfg)
	tn.Start(ctx)
	tn.Time().Sleep(time.Millisecond) // wait for route generation and dissemination

	payload := make([]byte, 300e3) // 300KB
	for i := range payload {
		payload[i] = byte(i)
	}

	// send from V0
	go func() {
		err := tn.dogwood[0].Send(ctx, payload)
		if err != nil {
			t.Error(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	// receive at V1
	go func() {
		defer wg.Done()
		p, v, err := tn.Dogwood(1).Receive(ctx)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(p, payload) {
			t.Errorf("unexpected payload")
		}
		if v != tn.VID(0) {
			t.Errorf("unexpected source")
		}
	}()

	// receive at V2
	go func() {
		defer wg.Done()
		p, v, err := tn.Dogwood(2).Receive(ctx)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(p, payload) {
			t.Errorf("unexpected payload")
		}
		if v != tn.VID(0) {
			t.Errorf("unexpected source")
		}
	}()

	wg.Wait()
}
