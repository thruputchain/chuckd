// Package encoding converts public identifier bytes without changing their value.
package encoding

import (
	"fmt"

	"github.com/mr-tron/base58"
)

// Base58 encodes identifier bytes with the case-sensitive Bitcoin alphabet.
func Base58(value []byte) string { return base58.Encode(value) }

// Decode32 accepts exactly 32 bytes of unprefixed base58. The text bound avoids
// converting oversized input, and errors never include the supplied text.
func Decode32(text string) ([32]byte, error) {
	var result [32]byte
	if len(text) < 32 || len(text) > 44 {
		return result, fmt.Errorf("expected a 32-byte base58 identifier")
	}
	decoded, err := base58.Decode(text)
	if err != nil || len(decoded) != len(result) {
		return result, fmt.Errorf("expected a 32-byte base58 identifier")
	}
	copy(result[:], decoded)
	return result, nil
}
