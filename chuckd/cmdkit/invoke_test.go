package cmdkit

import (
	"context"
	"strings"
	"testing"
)

func TestPprofRates(t *testing.T) {
	oldEnabled, oldMutex, oldBlock := pprofEnabled, pprofMutexFraction, pprofBlockRate
	t.Cleanup(func() { SetPprof(oldEnabled, oldMutex, oldBlock) })
	for _, enabled := range []bool{false, true} {
		for _, rates := range [][2]int{{0, 0}, {5, 10000}} {
			SetPprof(enabled, rates[0], rates[1])
			mutexCalls, blockCalls := 0, 0
			applyPprofRates(func(rate int) int {
				mutexCalls++
				if rate != rates[0] {
					t.Fatalf("mutex rate = %d, want %d", rate, rates[0])
				}
				return 0
			}, func(rate int) {
				blockCalls++
				if rate != rates[1] {
					t.Fatalf("block rate = %d, want %d", rate, rates[1])
				}
			})
			want := 0
			if enabled {
				want = 1
			}
			if mutexCalls != want || blockCalls != want {
				t.Fatalf("enabled=%v: setter calls = (%d, %d), want (%d, %d)", enabled, mutexCalls, blockCalls, want, want)
			}
		}
	}
}

func TestProfileRateConflicts(t *testing.T) {
	oldEnabled, oldMutex, oldBlock := pprofEnabled, pprofMutexFraction, pprofBlockRate
	oldMutexPath, oldBlockPath := mutexProfilePath, blockProfilePath
	t.Cleanup(func() {
		SetPprof(oldEnabled, oldMutex, oldBlock)
		SetMutexProfilePath(oldMutexPath)
		SetBlockProfilePath(oldBlockPath)
	})
	for _, tc := range []struct {
		name                 string
		enabled              bool
		mutexPath, blockPath string
		mutexRate, blockRate int
		conflict             string
	}{
		{"mutex conflict", true, "mutex.pprof", "", 5, 0, "--mutex"},
		{"block conflict", true, "", "block.pprof", 0, 10000, "--block"},
		{"default rates", true, "mutex.pprof", "block.pprof", 0, 0, ""},
		{"live only", true, "", "", 5, 10000, ""},
		{"disabled", false, "mutex.pprof", "block.pprof", 5, 10000, ""},
		{"independent mutex", true, "mutex.pprof", "", 0, 10000, ""},
		{"independent block", true, "", "block.pprof", 5, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			SetPprof(tc.enabled, tc.mutexRate, tc.blockRate)
			SetMutexProfilePath(tc.mutexPath)
			SetBlockProfilePath(tc.blockPath)
			if tc.conflict == "" {
				if err := validateProfileOptions(); err != nil {
					t.Fatal(err)
				}
				return
			}
			called := false
			err := Invoke(context.Background(), func() { called = true })
			if err == nil || !strings.Contains(err.Error(), tc.conflict) || !strings.Contains(err.Error(), "--pprof-") {
				t.Fatalf("unexpected conflict error: %v", err)
			}
			if called {
				t.Fatal("work ran despite conflicting profile options")
			}
		})
	}
}
