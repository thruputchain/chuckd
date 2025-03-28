package coord

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"time"

	proto "github.com/groundhogchain/protocol-go"
)

func GenerateUPID(seed uint64) (proto.UPID, error) {
	var result proto.UPID

	// Get process ID
	pid := int32(os.Getpid())

	// Get current time in nanoseconds
	timestamp := time.Now().UnixNano()

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return result, fmt.Errorf("get hostname: %w", err)
	}

	// Pack all data into a byte slice
	data := make([]byte, 0, 32)
	buf := make([]byte, 8)

	binary.LittleEndian.PutUint32(buf[:4], uint32(pid))
	data = append(data, buf[:4]...)

	binary.LittleEndian.PutUint64(buf, uint64(timestamp))
	data = append(data, buf...)

	data = append(data, []byte(hostname)...)

	binary.LittleEndian.PutUint64(buf, uint64(seed))
	data = append(data, buf...)

	hash := sha256.Sum256(data)

	copy(result[:], hash[:8])
	return result, nil
}
