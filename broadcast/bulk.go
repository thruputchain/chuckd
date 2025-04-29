package broadcast

import (
	"context"
	"crypto/sha256"
	"fmt"
)

func (s *broadcastAPI) BulkSend(ctx context.Context, bulk []byte) (Channel, error) {

	chunkSize := s.svc.cfg.ChunkSize
	batchCount := (len(bulk) + chunkSize - 1) / chunkSize
	if batchCount != int(int32(batchCount)) {
		panic("too many chunks")
	}

	batchHash := sha256.Sum256(bulk)

	sch := s.makeChannel(ctx, int64(batchCount), int64(len(bulk)), batchHash)
	defer sch.Close()

	for i := 0; i < int(batchCount); i++ {
		err := sch.Send(ctx, bulk[i*int(chunkSize):min(len(bulk), (i+1)*int(chunkSize))])
		if err != nil {
			return sch.Channel(), err
		}
	}

	return sch.Channel(), nil
}

func (s *broadcastAPI) BulkReceive(ctx context.Context) (Channel, []byte, error) {

	rch := <-s.svc.bulkAcceptCh
	defer rch.Close()

	var payload []byte

	// accept discriminator sees batch_size and can thus prevent simple dos attacks
	for i := 0; i < int(rch.Channel().Batch_count); i++ {
		seqNo, _, chunk, err := rch.Receive(ctx)
		if err != nil {
			return rch.Channel(), nil, err
		}
		if seqNo != SeqNo(i) {
			return rch.Channel(), nil, fmt.Errorf("batch broadcast missing or out of order message")
		}
		payload = append(payload, chunk...)
	}

	return rch.Channel(), payload, nil
}
