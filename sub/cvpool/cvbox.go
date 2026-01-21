package cvpool

import (
	"context"

	"github.com/groundhogchain/chuckd/util/protoutil"
	"github.com/groundhogchain/lib-go/must"
	proto "github.com/groundhogchain/protocol-go"
)

type cvBox struct {
	cv    *proto.ConsensusValue
	ns    *protoutil.NominationSerialized
	ready <-chan *protoutil.NominationSerialized
}

func NewCVBox(ctx context.Context, cv proto.ConsensusValue) *cvBox {
	ready := make(chan *protoutil.NominationSerialized, 1)
	x := &cvBox{
		cv:    &cv,
		ns:    nil,
		ready: ready,
	}
	go func() {
		ns, err := protoutil.SerializeNomination(&cv.Nomination)
		must.NoError(context.WithoutCancel(ctx), err)
		ready <- &ns
		close(ready)
	}()
	return x
}

func (x *cvBox) CV() *proto.ConsensusValue {
	return x.cv
}

func (x *cvBox) NS() *protoutil.NominationSerialized {
	if ns, ok := <-x.ready; ok {
		x.ns = ns
		return ns
	} else {
		if x.ns == nil {
			panic("bug")
		}
		return x.ns
	}
}
