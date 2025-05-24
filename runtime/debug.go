package runtime

import (
	"bytes"
	"fmt"
	"runtime"
)

func ResourceUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var w bytes.Buffer
	fmt.Fprintf(&w, "Go=%d ", runtime.NumGoroutine())
	fmt.Fprintf(&w, "Alloc=%dKB ", bToKb(m.Alloc))
	fmt.Fprintf(&w, "TotalAlloc=%dKB ", bToKb(m.TotalAlloc))
	fmt.Fprintf(&w, "Sys=%dKB ", bToKb(m.Sys))
	fmt.Fprintf(&w, "NumGC=%v", m.NumGC)
	return w.String()
}

func bToKb(b uint64) uint64 {
	return b / 1024
}
