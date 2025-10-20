package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/groundhogchain/jay/lib/must"
	proto "github.com/groundhogchain/protocol-go"
)

// consensus values

func (x *DiskStorage) ConsensusValueRootpath() string {
	return path.Join(x.dir, "consensus")
}

func (x *DiskStorage) ConsensusValueFilepath(stamp time.Time, slot int64) string {
	year := fmt.Sprintf("%04d", stamp.Year())
	month := fmt.Sprintf("%02d", stamp.Month())
	day := fmt.Sprintf("%02d", stamp.Day())
	hour := fmt.Sprintf("%02d", stamp.Hour())
	minute := fmt.Sprintf("%02d", stamp.Minute())
	return path.Join(
		x.ConsensusValueRootpath(),
		year, month, day, hour, minute,
		fmt.Sprintf("consensus-%020d.xdr", slot),
	)
}

func ParseConsensusValueFilename(name string) (int64, error) {
	const prefix = "consensus-"
	const suffix = ".xdr"

	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return 0, fmt.Errorf("invalid filename format: %q", name)
	}

	numStr := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	slot, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid slot number in filename %q: %w", name, err)
	}

	return slot, nil
}

// scp envelopes

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

// Matches e.g. scp-000000017213.123456789-deadbeef.xdr
var scpRE = regexp.MustCompile(`^scp-(\d{12}\.\d{9})-([0-9a-fA-F]+)\.xdr$`)

func ListScpFiles(dir string) ([]string, error) {

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if scpRE.MatchString(name) {
			matches = append(matches, filepath.Join(dir, name))
		}
	}

	// list in ascending chrono order, breaking ties with hashes
	sort.Slice(matches, func(i int, j int) bool { return matches[i] < matches[j] })

	return matches, nil
}

func (x *DiskStorage) SCPEnvelopeSlotpath(slot proto.SCPSlotIndex) string {
	return path.Join(
		x.dir,
		"scp",
		fmt.Sprintf("%020%d", slot),
	)
}

func (x *DiskStorage) SCPEnvelopeFilepath(slot proto.SCPSlotIndex, stamp time.Time, hash []byte) string {
	return path.Join(
		x.SCPEnvelopeSlotpath(slot),
		fmt.Sprintf("scp-%s-%x.xdr", FormatUnixNano(stamp), hash),
	)
}

func (x *DiskStorage) CatchupBlockFilepath(height int64) string {
	return path.Join(
		x.dir,
		"catchup",
		fmt.Sprintf("catchup-%020d.xdr", height),
	)
}

func FilenameNonce(ctx context.Context) string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	must.NoError(ctx, err)
	return hex.EncodeToString(b)
}
