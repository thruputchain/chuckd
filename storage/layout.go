package storage

import (
	"fmt"
	"path"
	"time"
)

// all externalized consensus values
// recently emitted scp envelopes
// catchup blocks for recently committed blocks

type Layout struct {
	Dir string
}

func (x *Layout) ExternalizedFilepath(stamp time.Time, slot int64) string {
	year := fmt.Sprintf("Y%04d", stamp.Year())
	month := fmt.Sprintf("M%02d", stamp.Month())
	day := fmt.Sprintf("D%02d", stamp.Day())
	hour := fmt.Sprintf("H%02d", stamp.Hour())
	minute := fmt.Sprintf("m%02d", stamp.Minute())
	return path.Join(
		x.Dir,
		"externalized",
		year, month, day, hour, minute,
		fmt.Sprintf("externalized-%20d.xdr", slot),
	)
}

func (x *Layout) SCPEnvelopeFilepath(slot int64, seqno int) string {
	return path.Join(
		x.Dir,
		"scp",
		fmt.Sprintf("scp-%20d-%10d.xdr", slot, seqno),
	)
}

func (x *Layout) CatchupFilepath(height int64) string {
	return path.Join(
		x.Dir,
		"catchup",
		fmt.Sprintf("catchup-%20d.xdr", height),
	)
}
