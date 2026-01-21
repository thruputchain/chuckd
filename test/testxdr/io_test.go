package benchxdr

import (
	"os"
	"testing"
)

// BenchmarkWrite30MB tests how long it takes to write 30MB to disk
func BenchmarkWrite30MB(b *testing.B) {
	const dataSize = 30 * 1024 * 1024 // 30MB
	data := make([]byte, dataSize)

	// Fill with some data
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.SetBytes(dataSize)

	for b.Loop() {
		// Create temp file
		f, err := os.CreateTemp("", "benchmark-*.dat")
		if err != nil {
			b.Fatal(err)
		}
		filename := f.Name()

		// Write data
		_, err = f.Write(data)
		if err != nil {
			f.Close()
			os.Remove(filename)
			b.Fatal(err)
		}

		// Close and cleanup
		f.Close()
		os.Remove(filename)
	}
}
