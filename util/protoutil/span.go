package protoutil

import (
	"sort"

	"golang.org/x/exp/constraints"
)

type Span[T constraints.Ordered] struct {
	From T
	To   T
}

func (s Span[T]) IsNotEmpty() bool {
	return s.From < s.To
}

type Spans[T constraints.Ordered] []Span[T]

func (ss Spans[T]) IsNotEmpty() bool {
	return len(ss) > 0 // assumes no empty spans inside
}

func (ss Spans[T]) Len() int           { return len(ss) }
func (ss Spans[T]) Less(i, j int) bool { return ss[i].From < ss[j].From }
func (ss Spans[T]) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }
func (ss Spans[T]) Sort()              { sort.Sort(ss) }

func SumSpans[T constraints.Ordered](spans ...Span[T]) (sum Spans[T]) {
	if len(spans) == 0 {
		return nil
	}
	Spans[T](spans).Sort()
	open := spans[0]
	for _, span := range spans[1:] {
		switch {
		case span.From <= open.To:
			open.To = max(open.To, span.To)
		case span.From > open.To:
			sum = append(sum, open)
			open = span
		}
	}
	if open.IsNotEmpty() {
		sum = append(sum, open)
	}
	return sum
}
