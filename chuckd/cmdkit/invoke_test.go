package cmdkit

import "testing"

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
