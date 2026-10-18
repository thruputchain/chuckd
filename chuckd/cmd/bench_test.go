package cmd

import (
	"strings"
	"testing"

	"github.com/thruputchain/chuckd/util/encoding"
)

func TestBenchmarkNetworkHashDefaultPreservesBytes(t *testing.T) {
	text := benchCoordinatorLoadgenCmd.Flags().Lookup("network-hash").DefValue
	got, err := encoding.Decode32(text)
	if err != nil || got != [32]byte{31: 1} {
		t.Fatalf("benchmark network identity changed: %x, %v", got, err)
	}
}

func TestBenchmarkRejectsInvalidNetworkHashBeforeStartup(t *testing.T) {
	original := benchNetworkHash
	t.Cleanup(func() { benchNetworkHash = original })
	for _, text := range []string{strings.Repeat("0", 63) + "1", "0x" + strings.Repeat("ab", 32), strings.Repeat("2", 32), strings.Repeat("1", 33)} {
		benchNetworkHash = text
		if err := benchCoordinatorLoadgenCmd.RunE(benchCoordinatorLoadgenCmd, nil); err == nil || !strings.Contains(err.Error(), "network hash:") {
			t.Fatalf("invalid network hash should fail before config loading, got %v", err)
		}
	}
}
