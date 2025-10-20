package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/groundhogchain/jay/lib/must"
)

// all externalized consensus values
// recently emitted scp envelopes
// catchup blocks for recently committed blocks

func (x *DiskStorage) ConsensusValueFilepath(stamp time.Time, slot int64) string {
	year := fmt.Sprintf("Y%04d", stamp.Year())
	month := fmt.Sprintf("M%02d", stamp.Month())
	day := fmt.Sprintf("D%02d", stamp.Day())
	hour := fmt.Sprintf("H%02d", stamp.Hour())
	minute := fmt.Sprintf("m%02d", stamp.Minute())
	return path.Join(
		x.dir,
		"consensus",
		year, month, day, hour, minute,
		fmt.Sprintf("consensus-%20d.xdr", slot),
	)
}

func FormatUnixNano(t time.Time) string {
	seconds := t.Unix()
	nanos := t.Nanosecond()
	return fmt.Sprintf("%012d.%09d", seconds, nanos)
}

func ParseUnixNano(s string) (time.Time, error) {
	parts := strings.SplitN(s, ".", 2)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid format: %q", s)
	}

	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid seconds: %v", err)
	}

	nano, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid nanoseconds: %v", err)
	}

	if nano < 0 || nano >= 1e9 {
		return time.Time{}, fmt.Errorf("nanoseconds out of range: %d", nano)
	}

	return time.Unix(sec, nano).UTC(), nil
}

func (x *DiskStorage) SCPEnvelopeFilepath(stamp time.Time, seqno int) string {
	return path.Join(
		x.dir,
		"scp",
		fmt.Sprintf("scp-%s-%10d.xdr", FormatUnixNano(stamp), seqno),
	)
}

func (x *DiskStorage) CatchupBlockFilepath(height int64) string {
	return path.Join(
		x.dir,
		"catchup",
		fmt.Sprintf("catchup-%20d.xdr", height),
	)
}

func FilenameNonce(ctx context.Context) string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	must.NoError(ctx, err)
	return hex.EncodeToString(b)
}
