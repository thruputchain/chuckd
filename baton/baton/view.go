package baton

import (
	"bytes"
	"fmt"
	"time"

	"github.com/groundhogchain/chuckd/baton/batonkit"
	proto "github.com/groundhogchain/protocol-go"
)

type View struct {
	Stamp        time.Time
	Blocks       []ViewBlock
	Reqs         [][]batonkit.ID64 // catchup requests already fired for this block
	NextToCommit *batonkit.ValidatableBlock
}

func (v *View) TipToNominateOn() *batonkit.Tip {
	if len(v.Blocks) != 1 {
		return nil
	}
	db := v.Blocks[0].(ViewDbBlock)
	return &db.Tip
}

func (v *View) NextBlockToCommit() *batonkit.ValidatableBlock {
	return v.NextToCommit
}

func (v *View) nextBlockToCommit(b *Baton) *batonkit.ValidatableBlock {

	if len(v.Blocks) < 2 || !v.Blocks[1].IsReadyToValidate() {
		return nil
	}
	vb := v.Blocks[1].ValidatableBlock()

	// db state
	db := v.Blocks[0].(ViewDbBlock)
	vb.Onto = db.Tip

	// are we committing a block that we have already validated?
	if vtd, ok := b.validated[vb.BlockHash()]; ok {
		vb.AlreadyValidated = &vtd.BlockID
	}

	return vb
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
			case ViewExternalizedBlock:
				if u.IsReadyToValidate() {
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
