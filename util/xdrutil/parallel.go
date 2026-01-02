package xdrutil

import (
	"bytes"
	"sync"
	"sync/atomic"

	"github.com/groundhogchain/chuckd/util"
	"github.com/groundhogchain/lib-go/libmem"
	"github.com/groundhogchain/lib-go/libxdr"
	proto "github.com/groundhogchain/protocol-go"
	"github.com/xdrpp/goxdr/xdr"
)

func marshalParallel(
	sectionM libxdr.XdrMarshaler,
	nGo int,
	tokens []proto.SignedTransactionHash,
) proto.ParallelSignedTransactionHashBatch {

	// determine number of sections

	nTokens := len(tokens)
	nTokensPerGo := util.CeilDivSigned(nTokens, nGo)       // tokens per goroutine = num tokens / num goroutines
	nSections := util.CeilDivSigned(nTokens, nTokensPerGo) // num useful goroutines = num tokens / num tokens per goroutine

	index := make([]uint32, nSections)
	sections := make([]libmem.BorrowedBuffer, nSections)
	var ntotal atomic.Uint32

	// marshal sections

	var wg sync.WaitGroup
	wg.Add(nSections)
	for i := range nSections {
		go func() {
			defer wg.Done()

			lo := i * nTokensPerGo
			hi := min((i+1)*nTokensPerGo, nTokens)

			sections[i] = sectionM.Borrow()
			xout := xdr.XdrOut{Out: sections[i]}

			for _, token := range tokens[lo:hi] {
				xout.Marshal("", proto.XDR_SignedTransactionHash(&token))
			}

			k := uint32(sections[i].Len())
			index[i] = k
			ntotal.Add(k)
		}()
	}
	wg.Wait()

	// merge sections

	xbuf := bytes.NewBuffer(make([]byte, 0, int(ntotal.Load())))
	for _, section := range sections {
		xbuf.Write(section.Bytes())
		section.Recycle()
	}

	return proto.ParallelSignedTransactionHashBatch{
		Index: index,
		Xdr:   xbuf.Bytes(),
	}
}

// 1'000'000 xs / 20 = 50'000 xs per go
// 1'000'000 xs / 50'000 xs per go = 20 go

// 2 xs / 10 go = 1 xs per go
// 2 xs / 1 xs per go = 2 go
