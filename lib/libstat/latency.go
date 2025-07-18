package libstat

import (
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/lib/liblog"
	"github.com/groundhogchain/chuckd/lib/libtime"
)

type LatencyMeter struct {
	time libtime.Time
	ctr  Counter
}

type LatencyMeterSpan struct {
	meter *LatencyMeter
	start time.Time
}

func NewLatency(time_ libtime.Time) *LatencyMeter {
	return &LatencyMeter{
		time: time_,
	}
}

func NewLatencyWithReport(time_ libtime.Time, label string, freq time.Duration) *LatencyMeter {
	d := NewLatency(time_)
	go func() {
		for {
			time_.Sleep(freq)
			liblog.Infof("%s latency %s", label, d.String())
		}
	}()
	return d
}

func (x *LatencyMeter) Start() LatencyMeterSpan {
	return LatencyMeterSpan{
		meter: x,
		start: x.time.Now(),
	}
}

func (x LatencyMeterSpan) Stop() {
	x.meter.Add(x.meter.time.Now().Sub(x.start))
}

func (x *LatencyMeter) Add(d time.Duration) {
	x.ctr.Add(float64(d.Nanoseconds()))
}

func (x *LatencyMeter) Average() time.Duration {
	return time.Duration(x.ctr.Average())
}

func (x *LatencyMeter) Min() time.Duration {
	return time.Duration(x.ctr.Min())
}

func (x *LatencyMeter) Max() time.Duration {
	return time.Duration(x.ctr.Max())
}

func (x *LatencyMeter) Stddev() time.Duration {
	return time.Duration(x.ctr.Stddev())
}

func (x *LatencyMeter) String() string {
	return fmt.Sprintf("%dnsec(±%dnsec) [%dnsec, %dnsec]",
		x.Average(), x.Stddev(), x.Min(), x.Max())
}
