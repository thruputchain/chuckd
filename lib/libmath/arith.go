package libmath

type Integral interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func DivideRoundUp[T Integral](dividend, divisor T) T {
	return (dividend + divisor - 1) / divisor
}
