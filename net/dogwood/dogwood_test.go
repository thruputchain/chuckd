package dogwood

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libstat"
	"github.com/groundhogchain/chuckd/runtime"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/rs/zerolog"
)

func TestMain(m *testing.M) {
	liblog.LogLevel(zerolog.InfoLevel)
	runtime.CaptureInterrupts(true)
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
		Endpoint_max_pending:       10,
		Endpoint_num_clients:       10,
		Endpoint_idle_ttl_msec:     60e3, // 1 minute
		Endpoint_redial_delay_msec: 100,  // 100 msec
		//
		Stream_receiver_worker_count: 128,
		Stream_sender_worker_count:   2,
		//
		Transmission_receiver_decode_worker_count: 128,
		Transmission_sender_worker_count:          2,
		//
		Transmission_partial_max_ttl_msec:  10e3, // 10 sec
		Transmission_partial_idle_ttl_msec: 1e3,  // 1 sec
	}

	tn := NewDogwood3TCPNet(cfg)
	tn.Start(ctx)
	tn.Time().Sleep(10 * time.Millisecond) // wait for route generation and dissemination

	payload := make([]byte, 300e3) // 300KB
	for i := range payload {
		payload[i] = byte(i)
	}

	const N = 10000

	// send from V0
	go func() {
		for i := 0; i < N; i++ {
			if i%1000 == 0 {
				liblog.Infof("%v resource usage: %s", tn.String(0), runtime.ResourceUsage())
				liblog.Infof("%v dogwood stats: %v", tn.String(0), tn.dogwood[0].Stats())
			}
			err := tn.dogwood[0].Send(ctx, payload)
			if err != nil {
				t.Error(err)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	numReceived := make([]int, 3) // 3 validators
	throughput := make([]libstat.Counter, 3)

	receiveFn := func(vIndex int, check bool) {
		defer wg.Done()
		j := 0
		for {
			if j%1000 == 0 {
				liblog.Infof("%v resource usage: %s", tn.String(vIndex), runtime.ResourceUsage())
				liblog.Infof("%v dogwood stats: %v", tn.String(vIndex), tn.dogwood[vIndex].Stats())
			}
			rCtx, rCancel := context.WithTimeout(ctx, 2*time.Second)
			p, v, err := tn.Dogwood(vIndex).Receive(rCtx)
			rCancel()
			// p, v, err := tn.Dogwood(vIndex).Receive(ctx)
			if errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if err != nil {
				t.Error(err)
			}
			if check && !bytes.Equal(p, payload) {
				t.Errorf("unexpected payload")
			}
			if check && v != tn.VID(0) {
				t.Errorf("unexpected source")
			}
			numReceived[vIndex] += 1
			throughput[vIndex].Add(float64(len(p)))
			j += 1
		}
	}

	// receive at V1
	go receiveFn(1, true)

	// receive at V2
	go receiveFn(2, true)

	wg.Wait()

	for vIndex := 1; vIndex < 3; vIndex++ {
		fmt.Printf("validator %d received %d transmission out of %d, throughput %v/sec\n",
			vIndex, numReceived[vIndex], N, humanize.Bytes(uint64(throughput[vIndex].SpeedPerSec())))
	}
}
