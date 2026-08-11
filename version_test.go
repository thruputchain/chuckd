package chuckd

import (
	"strings"
	"testing"
	"time"
)

func TestShortDistributionVersionPreservesComponentIdentity(t *testing.T) {
	lk.Lock()
	originalInfo := versionInfo
	versionInfo = VersionInfo{
		Version:    "v0.1.12",
		Revision:   "0123456789abcdef",
		LastCommit: time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC),
	}
	lk.Unlock()
	originalDistributionVersion := distributionVersion
	distributionVersion = "0.1.19-rc.1"
	t.Cleanup(func() {
		lk.Lock()
		versionInfo = originalInfo
		lk.Unlock()
		distributionVersion = originalDistributionVersion
	})

	got := Short()
	if !strings.HasPrefix(got, "0.1.19-rc.1 ") {
		t.Fatalf("Short() = %q, expected distribution version first", got)
	}
	if !strings.Contains(got, "component v0.1.12-rev-0123456-2026-08-10") {
		t.Fatalf("Short() = %q, expected component provenance", got)
	}
}

func TestShortWithoutDistributionVersionIsUnchanged(t *testing.T) {
	lk.Lock()
	originalInfo := versionInfo
	versionInfo = VersionInfo{Version: "v0.1.12"}
	lk.Unlock()
	originalDistributionVersion := distributionVersion
	distributionVersion = ""
	t.Cleanup(func() {
		lk.Lock()
		versionInfo = originalInfo
		lk.Unlock()
		distributionVersion = originalDistributionVersion
	})

	if got := Short(); got != "v0.1.12" {
		t.Fatalf("Short() = %q, want v0.1.12", got)
	}
}
