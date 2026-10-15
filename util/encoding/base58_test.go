package encoding

import (
	"strings"
	"testing"
)

func TestDecode32(t *testing.T) {
	for _, value := range [][32]byte{{}, {1}, {31: 1}, {0xff, 0xff}} {
		got, err := Decode32(Base58(value[:]))
		if err != nil || got != value {
			t.Fatalf("identifier bytes changed: %v", err)
		}
	}
	if Base58(make([]byte, 32)) != strings.Repeat("1", 32) {
		t.Fatal("zero bytes must retain leading ones")
	}
	for _, text := range []string{"", strings.Repeat("1", 31), strings.Repeat("1", 33), strings.Repeat("z", 44), strings.Repeat("0", 32), "0x" + strings.Repeat("ab", 32), strings.Repeat("a", 64)} {
		if _, err := Decode32(text); err == nil || err.Error() != "expected a 32-byte base58 identifier" {
			t.Fatalf("invalid identifier must return a redacted error: %v", err)
		}
	}
}
