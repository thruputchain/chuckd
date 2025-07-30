package libmem

func LinkBorrow[P any, Q any](p BorrowedValue[P], q Q) BorrowedValue[Q] {
	return &linkedBorrow[P, Q]{linked: p, v: q}
}

type linkedBorrow[P any, Q any] struct {
	linked BorrowedValue[P]
	v      Q
}

func (x *linkedBorrow[P, Q]) Value() Q {
	return x.v
}

func (x *linkedBorrow[P, Q]) Recycle() {
	x.linked.Recycle()
}
