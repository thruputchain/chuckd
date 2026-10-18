package protoutil

import (
	"time"

	proto "github.com/thruputchain/thruput-protocol-go"
)

func TimepointFromTime(t time.Time) proto.TimePoint {
	return uint64(t.UnixNano())
}

func TimeFromTimepoint(p proto.TimePoint) time.Time {
	return time.Unix(0, int64(p))
}
