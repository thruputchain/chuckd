package util

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// CeilDiv performs integer division that always rounds up
func CeilDivSigned[T Signed](a, b T) T {
	if b == 0 {
		panic("division by zero")
	}

	// Handle negative numbers appropriately
	if (a > 0 && b > 0) || (a < 0 && b < 0) {
		// Same sign: use (a + b - 1) / b
		return (a + b - 1) / b
	} else {
		// Different signs: regular division already rounds toward zero
		return a / b
	}
}
