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

	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

// consensus values

func (x *DiskStorage) ConsensusValuesDirpath() string {
	return path.Join(x.dir, "consensus")
}

func SlotToParts(h int64) [3]string {
	return [3]string{
		fmt.Sprintf("%03d", h/10e9),
		fmt.Sprintf("%03d", (h%10e9)/10e6),
		fmt.Sprintf("%03d", (h%10e6)/10e3),
	}
}

func (x *DiskStorage) ConsensusValueFilepath(slot int64) string {
	parts := SlotToParts(slot)
	return path.Join(
		x.ConsensusValuesDirpath(),
		parts[0], parts[1], parts[2],
		fmt.Sprintf("consensus-%012d.xdr", slot),
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

func (x *DiskStorage) SCPEnvelopesDirpath() string {
	return path.Join(
		x.dir,
		"scp",
	)
}

func (x *DiskStorage) SCPEnvelopeSlotpath(slot proto.SCPSlotIndex) string {
	return path.Join(
		x.SCPEnvelopesDirpath(),
		fmt.Sprintf("%012d", slot),
	)
}

func (x *DiskStorage) SCPEnvelopeFilepath(slot proto.SCPSlotIndex, stamp time.Time, hash []byte) string {
	return path.Join(
		x.SCPEnvelopeSlotpath(slot),
		fmt.Sprintf("scp-%s-%x.xdr", FormatUnixNano(stamp), hash),
	)
}

func (x *DiskStorage) CatchupBlocksDirpath() string {
	return path.Join(
		x.dir,
		"catchup",
	)
}

func (x *DiskStorage) CatchupBlockFilepath(height proto.BlockHeight, hash proto.BlockHash) string {
	return path.Join(
		x.CatchupBlocksDirpath(),
		fmt.Sprintf("catchup-%012d-%x.xdr", height, hash),
	)
}

func FilenameNonce(ctx context.Context) string {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	must.NoError(ctx, err)
	return hex.EncodeToString(b)
}

// Matches e.g. catchup-000000000123-deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef.xdr
var catchupRE = regexp.MustCompile(`^catchup-([0-9]{12})-([0-9a-f]{64})\.xdr$`)

func ListCatchupFiles(dir string) ([]string, error) {

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
		if catchupRE.MatchString(name) {
			matches = append(matches, filepath.Join(dir, name))
		}
	}

	// list in ascending chrono order, breaking ties with hashes
	sort.Slice(matches, func(i int, j int) bool { return matches[i] < matches[j] })

	return matches, nil
}

func ListCatchupFilesAtHeight(dir string, height proto.BlockHeight) ([]string, error) {
	all, err := ListCatchupFiles(dir)
	if err != nil {
		return nil, err
	}

	prefix := fmt.Sprintf("catchup-%012d-", height)
	filtered := make([]string, 0, len(all))
	for _, full := range all {
		if strings.HasPrefix(filepath.Base(full), prefix) {
			filtered = append(filtered, full)
		}
	}

	return filtered, nil
}

// Matches strings like: "00000000000000001234"
var scpSlotRE = regexp.MustCompile(`^[0-9]{12}$`)

func ListSCPSlotDirs(dir string) ([]string, error) {

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var matches []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if scpSlotRE.MatchString(name) {
			matches = append(matches, filepath.Join(dir, name))
		}
	}

	// list in ascending order
	sort.Slice(matches, func(i int, j int) bool { return matches[i] < matches[j] })

	return matches, nil
}
