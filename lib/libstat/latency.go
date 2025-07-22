package libstat

import (
	"fmt"
	"time"

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

func (x *LatencyMeter) Start() LatencyMeterSpan {
	return LatencyMeterSpan{
		meter: x,
		start: x.time.Now(),
	}
}

func (x *LatencyMeter) StartAt(s time.Time) LatencyMeterSpan {
	return LatencyMeterSpan{
		meter: x,
		start: s,
	}
}

func (x LatencyMeterSpan) Stop() {
	x.meter.Add(x.meter.time.Now().Sub(x.start))
}

func (x *LatencyMeter) Add(d time.Duration) int {
	return x.ctr.Add(float64(d.Nanoseconds()))
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
	return fmt.Sprintf("%s ± %s [%s, %s]",
		libtime.FmtDur(x.Average()), libtime.FmtDur(x.Stddev()), libtime.FmtDur(x.Min()), libtime.FmtDur(x.Max()))
}

type LatencyStats struct {
	Summary string
}

func (x *LatencyMeter) Stats() LatencyStats {
	c := x.ctr.Stats()
	return LatencyStats{
		Summary: fmt.Sprintf("%s ± %s [%s : %s]",
			libtime.FmtDur(time.Duration(c.Average)),
			libtime.FmtDur(time.Duration(c.Stddev)),
			libtime.FmtDur(time.Duration(c.Min)),
			libtime.FmtDur(time.Duration(c.Max)),
		),
	}
}
