package loadgen

type StatFloat64 struct {
	sum   float64
	count int64
}

func NewStatFloat64() *StatFloat64 {
	return &StatFloat64{}
}

func (x *StatFloat64) Add(s float64) {
	x.sum += s
	x.count += 1
}

func (x *StatFloat64) Average() float64 {
	return x.sum / float64(x.count)
}
