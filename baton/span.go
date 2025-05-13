package baton

import (
	"sort"

	proto "github.com/groundhogchain/protocol-go"
)

type Span struct {
	From proto.BlockHeight
	To   proto.BlockHeight
}

func (s Span) IsNotEmpty() bool {
	return s.From < s.To
}

type Spans []Span

func (ss Spans) Len() int           { return len(ss) }
func (ss Spans) Less(i, j int) bool { return ss[i].From < ss[j].From }
func (ss Spans) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }
func (ss Spans) Sort()              { sort.Sort(ss) }

func SumSpans(spans ...Span) (sum Spans) {
	if len(spans) == 0 {
		return nil
	}
	Spans(spans).Sort()
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

