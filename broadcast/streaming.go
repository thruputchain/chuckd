package broadcast

import (
	"context"

	proto "github.com/groundhogchain/protocol-go"
)

func (s *broadcastAPI) ValidatorID() proto.ValidatorID {
	return s.svc.ValidatorID()
}

func (s *broadcastAPI) MakeChannel(ctx context.Context) SendChannel {
	return s.makeChannel(ctx, -1, 0, proto.BroadcastBatchHash{})
}

func (s *broadcastAPI) makeChannel(
	ctx context.Context,
	batchCount int64,
	batchSize int64,
	batchHash proto.BroadcastBatchHash,
) SendChannel {

	s.svc.lk.Lock()
	defer s.svc.lk.Unlock()

	channelID := s.svc.generateChannelID()
	treeNodes, children, treeHash := s.svc.buildMulticastTree(channelID)
	ch := Channel{
		Source_id:   s.svc.ValidatorID(),
		Channel_id:  channelID,
		Tree_hash:   treeHash,
		Batch_count: batchCount,
		Batch_size:  batchSize,
		Batch_hash:  batchHash,
	}
	sch := newSending(ctx, s.svc, ch, treeNodes, children)

	s.svc.sending[ch] = sch
	s.svc.trees.Ref(treeHash, treeNodes)

	return sch
}

func (s *broadcastAPI) AcceptChannel() ReceiveChannel {
	return <-s.svc.streamingAcceptCh
}
