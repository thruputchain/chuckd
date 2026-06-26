package baton

import (
	"bytes"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/sub/baton/batonkit"
	proto "github.com/groundhogchain/protocol-go"
)

type View struct {
	Stamp  time.Time
	Blocks []ViewBlock
	Reqs   [][]batonkit.ID64 // catchup requests already fired for this block
}

func (v *View) HighestExternalizedOrCommitted() proto.BlockHeight {
	return v.Blocks[len(v.Blocks)-1].BlockHeight()
}

func (v *View) NominateOnto() (*proto.DbTip, *externalizedBasis) {
	if len(v.Blocks) != 1 {
		return nil, nil
	}
	db := v.Blocks[0].(ViewDbBlock)
	return &db.TipDb, &db.TipExt
}

func (v *View) NextBlockToCommit() *batonkit.CommittableBlock {

	if len(v.Blocks) < 2 {
		return nil
	}

	committable := v.Blocks[1].Committable()

	if committable == nil {
		return nil
	}

	// db state
	db := v.Blocks[0].(ViewDbBlock)

	return &batonkit.CommittableBlock{
		Committable: committable,
		Onto:        db.TipDb,
	}
}

func (v *View) indexHeight(i int) proto.BlockHeight {
	return v.Blocks[0].BlockHeight() + proto.BlockHeight(i)
}

func (v *View) NeededUnrequestedSpans() Spans {
	spans := Spans{}
	var start *int
	for i := 1; i < len(v.Blocks); i++ {
		if len(v.Reqs[i]) > 0 {
			if start != nil {
				spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(i)})
				start = nil
			}
		} else {
			switch u := v.Blocks[i].(type) {
			case ViewMissing:
				if start == nil {
					j := i
					start = &j
				}
			case ViewExtConsensusBlock, ViewExtValueBlock:
				if u.Committable() != nil {
					if start != nil {
						spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(i)})
						start = nil
					}
				} else {
					if start == nil {
						j := i
						start = &j
					}
				}
			case ViewCatchupBlock:
				if start != nil {
					spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(i)})
					start = nil
				}
			default:
				panic("unreachable")
			}
		}
	}
	if start != nil {
		spans = append(spans, Span{From: v.indexHeight(*start), To: v.indexHeight(len(v.Blocks))})
		start = nil
	}
	return spans
}

func (v *View) ShortString() string {
	var w bytes.Buffer
	fmt.Fprintf(&w, "|")
	for _, b := range v.Blocks {
		fmt.Fprintf(&w, "%s|", b.Icon())
	}
	return w.String()
}
